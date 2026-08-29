package replication

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"rcloudstorage/internal/storage"
	"sort"
	"strings"
	"time"
)

const nodeTimeout = 5 * time.Second

// NodeClient knows how to call one storage node over HTTP.
type NodeClient struct {
	Addr string
	HTTP *http.Client
}

func NewNodeClient(addr string) *NodeClient {
	return &NodeClient{
		Addr: strings.TrimRight(addr, "/"),
		HTTP: &http.Client{
			Timeout: nodeTimeout,
		},
	}
}

// Coordinator is a StorageBackend which replicates data to several nodes.
type Coordinator struct {
	Nodes []*NodeClient
	W     int
	R     int
}

// This is a compile-time check. If any method has the wrong signature,
// Go will refuse to compile.
var _ storage.StorageBackend = (*Coordinator)(nil)

func NewCoordinator(nodes []*NodeClient, w, r int) (*Coordinator, error) {
	if len(nodes) == 0 {
		return nil, errors.New("coordinator needs at least one node")
	}

	if w < 1 || w > len(nodes) {
		return nil, fmt.Errorf("invalid W=%d for %d nodes", w, len(nodes))
	}

	if r < 1 || r > len(nodes) {
		return nil, fmt.Errorf("invalid R=%d for %d nodes", r, len(nodes))
	}

	return &Coordinator{
		Nodes: nodes,
		W:     w,
		R:     r,
	}, nil
}

// Put buffers once because one io.Reader cannot be read concurrently by
// multiple node requests. Each node gets its own bytes.Reader.
func (c *Coordinator) Put(key string, r io.Reader) error {

	// NOTE@mazidrehaan: Since coordinator node recieves only 1 chunk at a time
	// hence this ReadAll is safe.
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading object before replication: %w", err)
	}

	// NOTE@mazidrehaan: We use a buffered channel to avoid goroutine leaks. If we
	// used an unbuffered channel, then if the first W nodes acked, the rest of
	// the goroutines would block forever trying to send their result to the
	// channel. With a buffered channel, the goroutines can send their result and
	// exit, even if the main goroutine has already returned.
	results := make(chan error, len(c.Nodes))

	for _, node := range c.Nodes {
		node := node

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeTimeout)
			defer cancel()

			results <- node.put(ctx, key, data)
		}()
	}

	acked := 0

	// NOTE@mazidrehaan: We only wait for W successful acks. If we get W acks,
	// we return early and don't wait for the rest of the nodes to respond.
	for range c.Nodes {
		if err := <-results; err == nil {
			acked++

			if acked >= c.W {
				return nil
			}
		}
	}

	return fmt.Errorf(
		"write quorum failed: only %d/%d nodes acknowledged; need W=%d",
		acked,
		len(c.Nodes),
		c.W,
	)
}

// Get asks nodes concurrently. For this first version, all healthy replicas
// should contain identical bytes, so it returns the first successful result
// after obtaining R successful responses.
func (c *Coordinator) Get(key string) (io.ReadCloser, error) {
	type result struct {
		data []byte
		err  error
	}

	results := make(chan result, len(c.Nodes))

	for _, node := range c.Nodes {
		node := node

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeTimeout)
			defer cancel()

			data, err := node.get(ctx, key)
			results <- result{data: data, err: err}
		}()
	}

	successes := 0
	var firstData []byte

	for range c.Nodes {
		result := <-results

		if result.err != nil {
			continue
		}

		if successes == 0 {
			firstData = result.data
		}

		successes++

		if successes >= c.R {
			return io.NopCloser(bytes.NewReader(firstData)), nil
		}
	}

	return nil, fmt.Errorf(
		"read quorum failed: fewer than R=%d nodes returned %q",
		c.R,
		key,
	)
}

// Delete is also replicated. We require W successful deletions.
func (c *Coordinator) Delete(key string) error {
	results := make(chan error, len(c.Nodes))

	for _, node := range c.Nodes {
		node := node

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeTimeout)
			defer cancel()

			results <- node.delete(ctx, key)
		}()
	}

	acked := 0

	for range c.Nodes {
		if err := <-results; err == nil {
			acked++

			if acked >= c.W {
				return nil
			}
		}
	}

	return fmt.Errorf(
		"delete quorum failed: only %d/%d nodes acknowledged; need W=%d",
		acked,
		len(c.Nodes),
		c.W,
	)
}

// List asks every currently reachable node and returns the union of their keys.
// This is intentionally not quorum-based yet; reconciliation comes later.
func (c *Coordinator) List(prefix string) ([]string, error) {
	type result struct {
		keys []string
		err  error
	}

	results := make(chan result, len(c.Nodes))

	for _, node := range c.Nodes {
		node := node

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeTimeout)
			defer cancel()

			keys, err := node.list(ctx, prefix)
			results <- result{keys: keys, err: err}
		}()
	}

	unique := make(map[string]struct{})
	successes := 0

	for range c.Nodes {
		result := <-results
		if result.err != nil {
			continue
		}

		successes++

		for _, key := range result.keys {
			unique[key] = struct{}{}
		}
	}

	if successes == 0 {
		return nil, errors.New("listing failed: no nodes responded")
	}

	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys, nil
}

func (n *NodeClient) put(ctx context.Context, key string, data []byte) error {
	requestURL := n.rawObjectURL(key)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		requestURL,
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}

	resp, err := n.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("node %s returned %s", n.Addr, resp.Status)
	}

	return nil
}

func (n *NodeClient) get(ctx context.Context, key string) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		n.rawObjectURL(key),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := n.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, storage.ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("node %s returned %s", n.Addr, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (n *NodeClient) delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		n.rawObjectURL(key),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := n.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("node %s returned %s", n.Addr, resp.Status)
	}

	return nil
}

func (n *NodeClient) list(ctx context.Context, prefix string) ([]string, error) {
	requestURL := n.Addr + "/internal/objects?prefix=" + url.QueryEscape(prefix)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		requestURL,
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := n.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("node %s returned %s", n.Addr, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return []string{}, nil
	}

	return strings.Split(text, "\n"), nil
}

func (n *NodeClient) rawObjectURL(key string) string {
	// PathEscape keeps a key as one URL path value rather than allowing
	// special URL characters to change the request.
	return n.Addr + "/internal/objects/" + url.PathEscape(key)
}
