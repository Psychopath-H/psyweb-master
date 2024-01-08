package psygo

import "strings"

type node struct {
	pattern  string  //待匹配路由，例如/p/:lang
	part     string  //路由中的一部分，例如:lang
	children []*node // 子结点，例如[doc, tutorial, intro]
	isWild   bool    //是否精确匹配，part含有 : 或 * 时为true
}

// matchChild 第一个成功匹配的节点，用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// matchChildren 所有匹配成功的节点，用于查找
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

// 对于路由来说，最重要的当然是注册与匹配了。开发服务时，注册路由规则，映射handler；访问时，匹配路由规则，查找到对应的handler。
// 因此，Trie 树需要支持节点的插入与查询。插入功能很简单，递归查找每一层的节点，如果没有匹配到当前part的节点，则新建一个，有一点需要注意，
// /p/:lang/doc只有在第三层节点，即doc节点，pattern才会设置为/p/:lang/doc。p和:lang节点的pattern属性皆为空。因此，当匹配结束时，
// 我们可以使用n.pattern == ""来判断路由规则是否匹配成功。例如，/p/python虽能成功匹配到:lang，但:lang的pattern值为空，因此匹配失败。

//查询功能，同样也是递归查询每一层的节点，退出规则是，匹配到了*，匹配失败，或者匹配到了第len(parts)层节点。

// insert 在前缀树中插入节点
func (n *node) insert(pattern string, parts []string, height int) {
	//插入到最后一个字符串了(根节点不算高度的情况下，树的高度和字符串长度相同)，把整个地址赋给当前节点
	if len(parts) == height {
		n.pattern = pattern
		return
	}
	//还没到最后一个字符串
	part := parts[height]
	child := n.matchChild(part)
	if child == nil { //没找到匹配的，那就新建一个
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	child.insert(pattern, parts, height+1) //对下一个节点做插入操作
}

// search 用于在前缀树中查询节点
func (n *node) search(parts []string, height int) *node {
	//找到最后的字符串了，或者说遇到通配符*了，要做返回操作
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}
	//没找到最后的字符串，继续去匹配
	part := parts[height]
	children := n.matchChildren(part)
	//对每一个能匹配上的children继续搜索
	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}
	return nil
}
