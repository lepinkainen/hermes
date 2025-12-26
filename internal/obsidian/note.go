package obsidian

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Note represents a complete markdown document with YAML frontmatter and body content.
type Note struct {
	Frontmatter *Frontmatter
	Body        string
}

// Frontmatter provides typed access to YAML frontmatter with sorted keys for deterministic output.
type Frontmatter struct {
	fields map[string]any
	keys   []string // Sorted key order for deterministic serialization
}

// NewFrontmatter creates a new empty Frontmatter.
func NewFrontmatter() *Frontmatter {
	return &Frontmatter{
		fields: make(map[string]any),
		keys:   []string{},
	}
}

// ParseMarkdown parses a markdown document with YAML frontmatter.
// Returns a Note with parsed frontmatter and body content.
// Missing frontmatter is valid and yields an empty Frontmatter.
func ParseMarkdown(content []byte) (*Note, error) {
	contentStr := string(content)

	// Check for frontmatter delimiters
	if !strings.HasPrefix(contentStr, "---\n") && !strings.HasPrefix(contentStr, "---\r\n") {
		// No frontmatter, entire content is body
		return &Note{
			Frontmatter: NewFrontmatter(),
			Body:        contentStr,
		}, nil
	}

	// Find the closing delimiter
	// Skip the opening "---"
	afterFirst := contentStr[3:]
	endIdx := strings.Index(afterFirst, "\n---\n")
	if endIdx == -1 {
		endIdx = strings.Index(afterFirst, "\r\n---\r\n")
		if endIdx == -1 {
			// No closing delimiter, treat as no frontmatter
			return &Note{
				Frontmatter: NewFrontmatter(),
				Body:        contentStr,
			}, nil
		}
		endIdx += 4 // account for \r\n
	}

	// Extract frontmatter and body
	frontmatterStr := afterFirst[:endIdx]
	bodyStartIdx := 3 + len(frontmatterStr) + 5 // "---" + frontmatter + "\n---\n"
	if bodyStartIdx > len(contentStr) {
		bodyStartIdx = len(contentStr)
	}
	body := strings.TrimPrefix(contentStr[bodyStartIdx:], "\n")

	// Parse YAML frontmatter
	var data map[string]any
	if err := yaml.Unmarshal([]byte(frontmatterStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Create Frontmatter with sorted keys
	fm := NewFrontmatter()
	for key, value := range data {
		fm.Set(key, value)
	}

	return &Note{
		Frontmatter: fm,
		Body:        body,
	}, nil
}

// Build serializes the Note back to markdown with YAML frontmatter.
// Tags are always written in flow-style format: [a, b, c]
// Frontmatter keys are sorted alphabetically for deterministic output.
func (n *Note) Build() ([]byte, error) {
	var buf bytes.Buffer

	// Write frontmatter if it has any fields
	if len(n.Frontmatter.keys) > 0 {
		buf.WriteString("---\n")

		frontmatterBytes, err := yaml.Marshal(n.Frontmatter)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
		}

		buf.Write(frontmatterBytes)
		buf.WriteString("---\n")
	}

	// Write body
	buf.WriteString(n.Body)

	return buf.Bytes(), nil
}

// Get retrieves a value from frontmatter.
func (f *Frontmatter) Get(key string) (any, bool) {
	val, ok := f.fields[key]
	return val, ok
}

// Set sets a value in frontmatter, maintaining sorted key order.
func (f *Frontmatter) Set(key string, value any) {
	_, exists := f.fields[key]
	f.fields[key] = value

	if !exists {
		// Insert in sorted position
		f.keys = append(f.keys, key)
		sort.Strings(f.keys)
	}
}

// Delete removes a key from frontmatter.
func (f *Frontmatter) Delete(key string) {
	delete(f.fields, key)
	for i, k := range f.keys {
		if k == key {
			f.keys = append(f.keys[:i], f.keys[i+1:]...)
			break
		}
	}
}

// GetString retrieves a string value, returning empty string if not found or wrong type.
func (f *Frontmatter) GetString(key string) string {
	val, ok := f.fields[key]
	if !ok {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetInt retrieves an int value, returning 0 if not found or wrong type.
func (f *Frontmatter) GetInt(key string) int {
	val, ok := f.fields[key]
	if !ok {
		return 0
	}
	if i, ok := val.(int); ok {
		return i
	}
	return 0
}

// GetBool retrieves a bool value, returning false if not found or wrong type.
func (f *Frontmatter) GetBool(key string) bool {
	val, ok := f.fields[key]
	if !ok {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// GetStringArray retrieves a string array, returning empty slice if not found or wrong type.
func (f *Frontmatter) GetStringArray(key string) []string {
	val, ok := f.fields[key]
	if !ok {
		return []string{}
	}
	return TagsFromAny(val)
}

// Keys returns a copy of the sorted frontmatter keys.
func (f *Frontmatter) Keys() []string {
	result := make([]string, len(f.keys))
	copy(result, f.keys)
	return result
}

// MarshalYAML implements custom YAML marshaling with sorted keys and flow-style tags.
func (f *Frontmatter) MarshalYAML() (interface{}, error) {
	// Create a mapping node with sorted key-value pairs
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 0, len(f.keys)*2),
	}

	for _, key := range f.keys {
		val := f.fields[key]

		// Create key node
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		}

		// Create value node with special handling for tags
		var valueNode *yaml.Node
		if key == "tags" {
			// Flow-style sequence: [a, b, c]
			tags := TagsFromAny(val)
			valueNode = &yaml.Node{
				Kind:  yaml.SequenceNode,
				Style: yaml.FlowStyle,
			}
			for _, tag := range tags {
				valueNode.Content = append(valueNode.Content, &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: tag,
				})
			}
		} else {
			// Normal value - encode as-is
			valueNode = &yaml.Node{}
			if err := valueNode.Encode(val); err != nil {
				return nil, err
			}
		}

		// Append key-value pair
		node.Content = append(node.Content, keyNode, valueNode)
	}

	return node, nil
}
