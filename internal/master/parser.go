package master

import (
	"fmt"
	"regexp"
	"strings"

	"container/antlr"
	cypher "container/internal/master/parsing"
)

var queries = []string{

	`CREATE (alice:Person {name: "Alice"}), (bob:Person {name: "Bob"}), (carol:Person {name: "Carol"}), (company:Company {name: "NeoTech"}), (alice)-[:FRIEND_WITH]->(bob), (carol)<-[:MENTORED_BY]-(alice), (bob)-[:COLLABORATES_WITH]->(carol), (company)-[:EMPLOYS]->(alice), (bob)-[:WORKS_AT]->(company)`,
	`MATCH
  (alice:Person {name: "Alice"})-[:FRIEND_WITH]->(bob:Person {name: "Bob"}),
  (alice)-[:MENTORED_BY]->(carol:Person {name: "Carol"}),
  (bob)-[:COLLABORATES_WITH]->(carol),
  (company:Company {name: "NeoTech"})-[:EMPLOYS]->(alice),
  (bob)-[:WORKS_AT]->(company)
RETURN alice, bob, carol, company
`,
	`MATCH
  (a:Person)-[:FRIEND_WITH]->(b:Person)-[:FRIEND_WITH]->(c:Person),
  (a)-[:MENTORED_BY]->(c:Person),
  (b)-[:COLLABORATES_WITH]->(c),
  (co:Company {name: "NeoTech"})-[:EMPLOYS]->(a),
  (b)-[:WORKS_AT]->(co)
RETURN a, b, c, co
`,
	`MATCH (p:Person)-[r1:WORKS_AT]->(c:Company),
      (c)-[r2:EMPLOYS]->(p)
DELETE r1, r2, p`,
}

type Subquery struct {
	BGPs         [][]string
	Matches      []string
	Creates      []string
	Conditions   []string
	WithClauses  []string
	ReturnClause string
	OrderLimit   string
	RawText      string
	Parts        []*SubqueryPart
}

type SubqueryPart struct {
	Type       string
	BGPs       [][]string
	Details    string
	Optional   bool
	Subqueries []*Subquery
}

type QueryTreeNode struct {
	Type     string
	Children []*QueryTreeNode
	Details  string
	Optional bool
	PhaseNum int
	BGPIndex int
	IsCase   bool
	OrderBy  []string
	Limit    string
	Create   string
	Match    string
}

type QueryCollector struct {
	*cypher.BaseCypherParserListener
	Subqueries      []*Subquery
	ruleNames       []string
	currentSubquery *Subquery
	currentPart     *SubqueryPart
	currentNode     *QueryTreeNode
	inUnion         bool
	insideCall      int
	subqueryAdded   bool
}

func (q *QueryCollector) EnterQueryCallSt(ctx *cypher.QueryCallStContext) {
	q.insideCall++
}

func (q *QueryCollector) ExitQueryCallSt(ctx *cypher.QueryCallStContext) {
	q.insideCall--
}

func (q *QueryCollector) EnterUnionSt(ctx *cypher.UnionStContext) {
	q.inUnion = true

	if q.currentSubquery != nil {
		q.Subqueries = append(q.Subqueries, q.currentSubquery)
		q.currentSubquery = nil
		q.subqueryAdded = false
	}
}

func (q *QueryCollector) ExitUnionSt(ctx *cypher.UnionStContext) {
	q.inUnion = false

	if q.currentSubquery != nil {
		q.Subqueries = append(q.Subqueries, q.currentSubquery)
		q.currentSubquery = nil
		q.subqueryAdded = true
	}
}

func (q *QueryCollector) processPattern(pat cypher.IPatternContext) {
	if pat == nil {
		return
	}

	parts := pat.AllPatternPart()
	for _, p := range parts {
		elem := p.PatternElem()
		bgp := buildPatternChain(elem)
		q.currentPart.BGPs = append(q.currentPart.BGPs, []string{bgp})
	}
}

func (q *QueryCollector) EnterMatchSt(ctx *cypher.MatchStContext) {
	if q.insideCall > 0 || q.inUnion {
		return
	}

	if q.currentSubquery == nil {
		q.currentSubquery = &Subquery{}
		q.subqueryAdded = false
	}

	part := &SubqueryPart{
		Type:     "MATCH",
		Optional: ctx.OPTIONAL() != nil,
	}
	q.currentSubquery.Parts = append(q.currentSubquery.Parts, part)
	q.currentPart = part
	if pattern := ctx.PatternWhere(); pattern != nil {
		if pat := pattern.Pattern(); pat != nil {
			q.processPattern(pat)
		}
	}
}

func (q *QueryCollector) EnterCreateSt(ctx *cypher.CreateStContext) {
	if q.insideCall > 0 {
		return
	}
	if q.currentSubquery == nil {
		q.currentSubquery = &Subquery{}
		q.subqueryAdded = false
	}
	part := &SubqueryPart{Type: "CREATE"}
	q.currentSubquery.Parts = append(q.currentSubquery.Parts, part)
	q.currentPart = part

	if pattern := ctx.Pattern(); pattern != nil {
		parts := pattern.AllPatternPart()
		for _, p := range parts {
			elem := p.PatternElem()
			bgp := buildPatternChain(elem)
			q.currentPart.BGPs = append(q.currentPart.BGPs, []string{bgp})
		}
	}

}

func (q *QueryCollector) ExitCreateSt(ctx *cypher.CreateStContext) {
	if !q.subqueryAdded && q.currentSubquery != nil {
		q.Subqueries = append(q.Subqueries, q.currentSubquery)
		q.subqueryAdded = true
	}
}

func (q *QueryCollector) EnterDeleteSt(ctx *cypher.DeleteStContext) {
	if q.insideCall > 0 {
		return
	}
	if q.currentSubquery == nil {
		q.currentSubquery = &Subquery{}
		q.subqueryAdded = false
	}

	var details strings.Builder
	if ctx.DETACH() != nil {
		details.WriteString("DETACH ")
	}
	details.WriteString("DELETE ")

	exprs := ctx.ExpressionChain().AllExpression()
	for i, expr := range exprs {
		if i > 0 {
			details.WriteString(", ")
		}
		details.WriteString(expr.GetText())
	}

	part := &SubqueryPart{
		Type:    "DELETE",
		Details: details.String(),
	}
	q.currentSubquery.Parts = append(q.currentSubquery.Parts, part)
	q.currentPart = part
}

func (q *QueryCollector) ExitDeleteSt(ctx *cypher.DeleteStContext) {
	if !q.subqueryAdded && q.currentSubquery != nil {
		q.Subqueries = append(q.Subqueries, q.currentSubquery)
		q.subqueryAdded = true
	}
}

func buildPatternChain(elem cypher.IPatternElemContext) string {
	var out strings.Builder

	if node := elem.NodePattern(); node != nil {
		out.WriteString(processNode(node))
	}
	for _, chain := range elem.AllPatternElemChain() {
		if rel := chain.RelationshipPattern(); rel != nil {
			out.WriteString(processRelationship(rel))
		}
		if node := chain.NodePattern(); node != nil {
			out.WriteString(processNode(node))
		}
	}
	return out.String()
}

func (q *QueryCollector) EnterWhere(ctx *cypher.WhereContext) {
	if q.currentSubquery == nil || q.insideCall > 0 {
		return
	}
	if expr := ctx.Expression(); expr != nil {
		q.currentSubquery.Conditions = append(q.currentSubquery.Conditions, expr.GetText())
	}
}

func (q *QueryCollector) EnterSubqueryExist(ctx *cypher.SubqueryExistContext) {
	if q.insideCall > 0 {
		return
	}

	part := &SubqueryPart{
		Type: "EXISTS",
	}

	if patternWhere := ctx.PatternWhere(); patternWhere != nil {
		part.Details = patternWhere.GetText()
	} else if regularQuery := ctx.RegularQuery(); regularQuery != nil {
		part.Details = regularQuery.GetText()
	}

	q.currentSubquery.Parts = append(q.currentSubquery.Parts, part)
	q.currentPart = part
}

func (q *QueryCollector) EnterWithSt(ctx *cypher.WithStContext) {
	if q.currentSubquery == nil {
		return
	}
	body := ctx.ProjectionBody()
	if body != nil {
		var items []string
		if pi := body.ProjectionItems(); pi != nil {
			projItems := pi.AllProjectionItem()
			for _, item := range projItems {
				items = append(items, item.GetText())
			}
		}
		q.currentSubquery.WithClauses = append(q.currentSubquery.WithClauses, strings.Join(items, ", "))
	}
}

func (q *QueryCollector) EnterReturnSt(ctx *cypher.ReturnStContext) {
	if q.currentSubquery == nil || q.insideCall > 0 {
		return
	}
	body := ctx.ProjectionBody()
	if body != nil {
		var items []string
		if pi := body.ProjectionItems(); pi != nil {
			projItems := pi.AllProjectionItem()
			for _, item := range projItems {
				items = append(items, item.GetText())
			}
		}
		q.currentSubquery.ReturnClause = strings.Join(items, ", ")

		if order := body.OrderSt(); order != nil {
			q.currentSubquery.OrderLimit += " ORDER BY " + order.GetText()
		}
		if limit := body.LimitSt(); limit != nil {
			q.currentSubquery.OrderLimit += " LIMIT " + limit.GetText()
		}
	}
	if !q.subqueryAdded {
		q.Subqueries = append(q.Subqueries, q.currentSubquery)
		q.subqueryAdded = true
	}
	q.currentSubquery = nil
}

func processNode(node cypher.INodePatternContext) string {
	var parts []string

	if variable := node.Symbol(); variable != nil {
		parts = append(parts, variable.GetText())
	}

	if labels := node.NodeLabels(); labels != nil {
		parts = append(parts, labels.GetText())
	}

	if props := node.Properties(); props != nil {
		parts = append(parts, props.GetText())
	}

	return "(" + strings.Join(parts, "") + ")"
}

func processRelationship(rel cypher.IRelationshipPatternContext) string {
	return rel.GetText()
}

func parseNestedQuery(queryText string) *QueryTreeNode {
	input := antlr.NewInputStream(queryText)
	lexer := cypher.NewCypherLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := cypher.NewCypherParser(tokens)
	tree := parser.Query()

	collector := &QueryCollector{
		BaseCypherParserListener: &cypher.BaseCypherParserListener{},
	}

	antlr.NewParseTreeWalker().Walk(collector, tree)

	if len(collector.Subqueries) > 0 {
		return collector.BuildSubTree(collector.Subqueries[0])
	}
	return &QueryTreeNode{Type: "EmptySubquery"}
}

func (q *QueryCollector) BuildQueryTree() *QueryTreeNode {
	root := &QueryTreeNode{Type: "Query"}
	if q.inUnion {
		root.Type = "UNION"
	}
	var unionRoot *QueryTreeNode
	for i, sq := range q.Subqueries {
		node := q.BuildSubTree(sq)

		if i > 0 && unionRoot != nil {

			unionNode := &QueryTreeNode{Type: "UNION"}
			unionNode.Children = append(unionNode.Children, root.Children[len(root.Children)-1])
			unionNode.Children = append(unionNode.Children, node)
			root.Children[len(root.Children)-1] = unionNode
		} else {
			root.Children = append(root.Children, node)
		}
	}
	return root
}

func (q *QueryCollector) BuildSubTree(sq *Subquery) *QueryTreeNode {
	root := &QueryTreeNode{Type: "Subquery"}

	for _, part := range sq.Parts {
		switch part.Type {
		case "CREATE":
			createNode := &QueryTreeNode{Type: "CREATE"}
			for idx, bgp := range part.BGPs {
				createNode.Children = append(createNode.Children, &QueryTreeNode{
					Type:    fmt.Sprintf("BGP %d", idx+1),
					Details: strings.Join(bgp, ", "),
				})
			}
			root.Children = append(root.Children, createNode)

		case "MATCH":
			matchNode := &QueryTreeNode{
				Type:     "MATCH",
				Optional: part.Optional,
			}
			for idx, bgp := range part.BGPs {
				matchNode.Children = append(matchNode.Children, &QueryTreeNode{
					Type:    fmt.Sprintf("BGP %d", idx+1),
					Details: strings.Join(bgp, ", "),
				})
			}
			root.Children = append(root.Children, matchNode)
		case "DELETE":
			delNode := &QueryTreeNode{
				Type:    "DELETE",
				Details: part.Details,
			}
			root.Children = append(root.Children, delNode)
		case "EXISTS":
			existsNode := &QueryTreeNode{
				Type:    "EXISTS",
				Details: part.Details,
			}
			if strings.Contains(part.Details, "MATCH") {
				nestedQuery := parseNestedQuery(part.Details)
				existsNode.Children = append(existsNode.Children, nestedQuery)
			}
			root.Children = append(root.Children, existsNode)
		}

	}
	whereGroup := &QueryTreeNode{Type: "WHERE"}
	for _, cond := range sq.Conditions {
		whereGroup.Children = append(whereGroup.Children, &QueryTreeNode{Type: "Condition", Details: cond})
	}
	if len(whereGroup.Children) > 0 {
		root.Children = append(root.Children, whereGroup)
	}

	for i, with := range sq.WithClauses {
		withNode := &QueryTreeNode{Type: "WITH", PhaseNum: i + 1}
		for _, item := range splitWithItems(with) {
			withNode.Children = append(withNode.Children, &QueryTreeNode{Type: "WithItem", Details: item})
		}
		root.Children = append(root.Children, withNode)
	}

	if len(sq.Conditions) > len(whereGroup.Children) {
		lastWhere := &QueryTreeNode{Type: "WHERE", Details: sq.Conditions[len(sq.Conditions)-1]}
		root.Children = append(root.Children, lastWhere)
	}

	if sq.ReturnClause != "" {
		retNode := &QueryTreeNode{Type: "RETURN"}
		for _, item := range splitReturnItems(sq.ReturnClause) {
			if strings.HasPrefix(strings.TrimSpace(item), "CASE") {
				caseNode := parseCaseExpression(item)
				retNode.Children = append(retNode.Children, caseNode)
			} else {
				retNode.Children = append(retNode.Children, &QueryTreeNode{Type: "ReturnItem", Details: item})
			}
		}

		if sq.OrderLimit != "" {
			orderBy, limit := extractOrderLimit(sq.OrderLimit)
			if len(orderBy) > 0 {
				orderNode := &QueryTreeNode{Type: "ORDER BY"}
				for _, o := range orderBy {
					orderNode.Children = append(orderNode.Children, &QueryTreeNode{Type: "OrderExpr", Details: o})
				}
				retNode.Children = append(retNode.Children, orderNode)
			}
			if limit != "" {
				limitNode := &QueryTreeNode{Type: "LIMIT", Details: limit}
				retNode.Children = append(retNode.Children, limitNode)
			}
		}
		root.Children = append(root.Children, retNode)
	}
	return root
}

func splitWithItems(withClause string) []string {
	var items []string
	for _, s := range strings.Split(withClause, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}
func splitReturnItems(ret string) []string {

	var items []string
	cur := ""
	inCase := false
	for _, s := range strings.Split(ret, ",") {
		if strings.Contains(s, "CASE") {
			inCase = true
		}
		if !inCase {
			items = append(items, strings.TrimSpace(s))
			continue
		}
		cur += s
		if strings.Contains(s, "END") {
			items = append(items, strings.TrimSpace(cur))
			inCase = false
			cur = ""
		} else {
			cur += ", "
		}
	}
	if cur != "" {
		items = append(items, cur)
	}
	return items
}
func parseCaseExpression(caseText string) *QueryTreeNode {

	node := &QueryTreeNode{Type: "CASE Expression", IsCase: true}
	inside := caseText
	if pos := strings.Index(inside, "CASE"); pos >= 0 {
		inside = inside[pos+4:]
	}
	if pos := strings.Index(inside, "END"); pos >= 0 {
		inside = inside[:pos]
	}
	for _, line := range strings.Split(inside, "WHEN") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "ELSE") {
			detail := strings.TrimPrefix(line, "ELSE")
			node.Children = append(node.Children, &QueryTreeNode{Type: "ELSE", Details: strings.TrimSpace(detail)})
		} else if strings.Contains(line, "THEN") {
			parts := strings.SplitN(line, "THEN", 2)
			cond := strings.TrimSpace(parts[0])
			res := strings.TrimSpace(parts[1])
			node.Children = append(node.Children, &QueryTreeNode{
				Type:    "WHEN",
				Details: fmt.Sprintf("%s THEN %s", cond, res),
			})
		}
	}
	return node
}
func extractOrderLimit(orderLimit string) ([]string, string) {
	orderBy := []string{}
	limit := ""
	ol := orderLimit
	if strings.Contains(ol, "ORDER BY") {
		orderStr := ol[strings.Index(ol, "ORDER BY")+len("ORDER BY"):]
		end := strings.Index(orderStr, "LIMIT")
		if end >= 0 {
			orderStr = orderStr[:end]
		}
		for _, o := range strings.Split(orderStr, ",") {
			orderBy = append(orderBy, strings.TrimSpace(o))
		}
	}
	if strings.Contains(ol, "LIMIT") {
		limit = strings.TrimSpace(ol[strings.Index(ol, "LIMIT")+len("LIMIT"):])
	}
	return orderBy, limit
}

type CypherListener struct {
	*cypher.BaseCypherParserListener
	ruleNames []string
	level     int
}

func NewCypherListener(ruleNames []string) *CypherListener {
	return &CypherListener{
		BaseCypherParserListener: &cypher.BaseCypherParserListener{},
		ruleNames:                ruleNames,
		level:                    0,
	}
}

func (l *CypherListener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	indent := strings.Repeat("  ", l.level)
	ruleName := l.ruleNames[ctx.GetRuleIndex()]
	fmt.Printf("%s%s: %s\n", indent+"  ", ruleName, ctx.GetText())
	l.level++
}

func (l *CypherListener) ExitEveryRule(ctx antlr.ParserRuleContext) {
	l.level--
}

func BuildQueryTrees(cypherQueries []string) ([]*QueryTreeNode, []*Subquery) {
	var queryTrees []*QueryTreeNode
	var allSubqueries []*Subquery

	for _, cypherInput := range cypherQueries {
		input := antlr.NewInputStream(cypherInput)
		lexer := cypher.NewCypherLexer(input)
		tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		parser := cypher.NewCypherParser(tokens)
		tree := parser.Query()

		collector := &QueryCollector{
			BaseCypherParserListener: &cypher.BaseCypherParserListener{},
			ruleNames:                parser.RuleNames,
			currentNode:              &QueryTreeNode{Type: "Query"},
		}

		antlr.NewParseTreeWalker().Walk(collector, tree)

		queryTree := collector.BuildQueryTree()
		queryTrees = append(queryTrees, queryTree)
		allSubqueries = append(allSubqueries, collector.Subqueries...)
	}

	return queryTrees, allSubqueries
}

func GenerateQueriesFromBGPs(subquery *Subquery) []string {
	var generatedQueries []string
	var matchPart, createPart, deletePart *SubqueryPart

	for _, part := range subquery.Parts {
		switch part.Type {
		case "MATCH":
			matchPart = part
		case "CREATE":
			createPart = part
		case "DELETE":
			deletePart = part
		}
	}

	hasDeleteOperation := deletePart != nil

	if matchPart != nil && createPart != nil {
		var query strings.Builder

		query.WriteString("MATCH ")
		if matchPart.Optional {
			query.WriteString("OPTIONAL ")
		}
		var bgpStrings []string
		for _, bgp := range matchPart.BGPs {
			bgpStrings = append(bgpStrings, strings.Join(bgp, ", "))
		}
		query.WriteString(strings.Join(bgpStrings, ", "))
		query.WriteString("\n")

		if len(subquery.Conditions) > 0 {
			query.WriteString("WHERE ")
			query.WriteString(strings.Join(subquery.Conditions, " AND "))
			query.WriteString("\n")
		}

		query.WriteString("CREATE ")
		var createBgpStrings []string
		for _, bgp := range createPart.BGPs {
			createBgpStrings = append(createBgpStrings, strings.Join(bgp, ", "))
		}
		query.WriteString(strings.Join(createBgpStrings, ", "))
		query.WriteString("\n")

		if hasDeleteOperation {
			query.WriteString(deletePart.Details)
			query.WriteString("\n")
		} else if subquery.ReturnClause != "" {
			query.WriteString("RETURN ")
			query.WriteString(subquery.ReturnClause)
			if subquery.OrderLimit != "" {
				query.WriteString(subquery.OrderLimit)
			}
			query.WriteString("\n")
		}

		generatedQueries = append(generatedQueries, query.String())
		return generatedQueries
	}

	for _, part := range subquery.Parts {
		switch part.Type {
		case "MATCH":
			for _, bgp := range part.BGPs {
				var query strings.Builder
				query.WriteString(part.Type)
				if part.Optional {
					query.WriteString(" OPTIONAL")
				}
				query.WriteString(" ")
				query.WriteString(strings.Join(bgp, ", "))
				query.WriteString("\n")

				if len(subquery.Conditions) > 0 {
					query.WriteString("WHERE ")
					query.WriteString(strings.Join(subquery.Conditions, " AND "))
					query.WriteString("\n")
				}

				if hasDeleteOperation {
					query.WriteString(deletePart.Details)
					query.WriteString("\n")
				} else {

					variables := []string{}
					for _, p := range bgp {
						variables = append(variables, extractBGPVariables(p)...)
					}
					seen := make(map[string]bool)
					uniqueVariables := []string{}
					for _, v := range variables {
						if _, ok := seen[v]; !ok {
							seen[v] = true
							uniqueVariables = append(uniqueVariables, v)
						}
					}

					if len(uniqueVariables) > 0 {
						query.WriteString("RETURN ")
						query.WriteString(strings.Join(uniqueVariables, ", "))
					} else {
						query.WriteString("RETURN 'match_found'")
					}
				}
				generatedQueries = append(generatedQueries, query.String())
			}

		case "CREATE":
			for _, bgp := range part.BGPs {
				var query strings.Builder
				query.WriteString(part.Type)
				query.WriteString(" ")
				query.WriteString(strings.Join(bgp, ", "))
				query.WriteString("\n")

				if hasDeleteOperation {
					query.WriteString(deletePart.Details)
					query.WriteString("\n")
				} else {

					variables := []string{}
					for _, p := range bgp {
						variables = append(variables, extractBGPVariables(p)...)
					}
					seen := make(map[string]bool)
					uniqueVariables := []string{}
					for _, v := range variables {
						if _, ok := seen[v]; !ok {
							seen[v] = true
							uniqueVariables = append(uniqueVariables, v)
						}
					}

					if len(uniqueVariables) > 0 {
						query.WriteString("RETURN ")
						query.WriteString(strings.Join(uniqueVariables, ", "))
					} else {
						query.WriteString("RETURN 'created'")
					}
				}
				generatedQueries = append(generatedQueries, query.String())
			}

		case "DELETE":

			if matchPart == nil && createPart == nil {
				query := fmt.Sprintf("%s", part.Details)
				generatedQueries = append(generatedQueries, query)
			}

		case "EXISTS":
			reMatch := regexp.MustCompile(`MATCH\s*(.*?)(?:\sWHERE.*)?$`)
			match := reMatch.FindStringSubmatch(part.Details)
			if len(match) > 1 {
				bgpPattern := strings.TrimSpace(match[1])

				variables := extractBGPVariables(bgpPattern)
				returnClause := ""
				if len(variables) > 0 {
					returnClause = "RETURN " + strings.Join(variables, ", ")
				} else {
					returnClause = "RETURN 'exists_check'"
				}
				generatedQueries = append(generatedQueries, fmt.Sprintf("MATCH %s\n%s", bgpPattern, returnClause))
			} else {
				generatedQueries = append(generatedQueries, fmt.Sprintf("CALL { %s } RETURN 'exists_evaluated'", part.Details))
			}
		}
	}
	return generatedQueries
}
