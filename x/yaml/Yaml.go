package yaml

import (
	"bufio"
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type Yaml struct {
	BaseDir string
}

func NewYaml(base_dir string) *Yaml {
	return &Yaml{base_dir}
}

// 处理 !include 开头标签, 替换到主文件
func (y *Yaml) prepare(filename string, indent int) ([]byte, error) { // {{{
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(y.BaseDir, filename)
	}

	// 打开文件
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result bytes.Buffer
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "!include ") {
			ind := strings.Index(line, "!include ")
			// 提取包含文件名
			//len("!include ") = 9
			includedFile := strings.TrimSpace(line[(ind + 9):])
			//fmt.Println(includedFile)
			// 递归处理包含文件
			includedContent, err := y.prepare(includedFile, indent+ind)
			if err != nil {
				return nil, fmt.Errorf("error processing included file %s: %v", includedFile, err)
			}
			result.Write(includedContent)
		} else {
			result.WriteString(strings.Repeat(" ", indent) + line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
} // }}}

func (y *Yaml) parse(filename string) ([]byte, error) { // {{{
	res, err := y.prepare(filename, 0)
	if err != nil {
		return nil, err
	}

	res, err = clean(res)
	if err != nil {
		return nil, err
	}

	res, err = y.process(res)
	if err != nil {
		return nil, err
	}

	return res, nil
} // }}}

// 去重
func clean(input []byte) ([]byte, error) { // {{{
	var node yaml.Node

	// 解析为yaml.Node以保留完整结构
	if err := yaml.Unmarshal(input, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// 处理根节点
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		deduplicateNodes(node.Content[0])
	}

	// 重新序列化
	output, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %v", err)
	}

	return output, nil
}

//}}}

func deduplicateNodes(node *yaml.Node) { // {{{
	if node.Kind == yaml.DocumentNode {
		for _, n := range node.Content {
			deduplicateNodes(n)
		}
		return
	}

	if node.Kind == yaml.MappingNode {
		// 使用map记录最后一次出现的位置
		lastIndex := make(map[string]int)
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}
			keyNode := node.Content[i]
			lastIndex[keyNode.Value] = i
		}

		// 构建新内容，只包含最后一次出现的key
		var newContent []*yaml.Node
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}

			keyNode := node.Content[i]
			if lastIndex[keyNode.Value] == i {
				valueNode := node.Content[i+1]
				newContent = append(newContent, keyNode, valueNode)
				deduplicateNodes(valueNode)
			}
		}

		node.Content = newContent
	}

	if node.Kind == yaml.SequenceNode {
		for _, n := range node.Content {
			deduplicateNodes(n)
		}
	}
} // }}}

// 处理值中include标签
func (y *Yaml) process(src []byte) ([]byte, error) { // {{{
	BaseDir := "."
	var node yaml.Node
	if err := yaml.Unmarshal(src, &node); err != nil {
		return nil, err
	}

	var processNode func(*yaml.Node) error
	processNode = func(n *yaml.Node) error {
		//		fmt.Printf("%#v", n)
		if n.Kind == yaml.DocumentNode || n.Kind == yaml.SequenceNode || n.Kind == yaml.MappingNode {
			for _, child := range n.Content {
				if err := processNode(child); err != nil {
					return err
				}
			}
		}

		if n.Tag == "!include" {
			var filename string
			if err := n.Decode(&filename); err != nil {
				return err
			}

			if !filepath.IsAbs(filename) {
				filename = filepath.Join(BaseDir, filename)
			}

			content, err := y.parse(filename)
			if err != nil {
				return err
			}

			var includedNode yaml.Node
			if err := yaml.Unmarshal(content, &includedNode); err != nil {
				return err
			}

			*n = *includedNode.Content[0]
		}
		return nil
	}

	if err := processNode(&node); err != nil {
		return nil, err
	}

	return yaml.Marshal(&node)
} // }}}

func (y *Yaml) YamlToMap(filename string) (map[string]any, error) { // {{{
	res, err := y.parse(filename)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := yaml.Unmarshal(res, &m); err != nil {
		return nil, fmt.Errorf("Error processing YAML: %v\n", err)
	}

	return m, nil
} // }}}
