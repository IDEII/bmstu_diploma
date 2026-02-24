package cypher

import "container/antlr"

type BaseCypherParserListener struct{}

var _ CypherParserListener = &BaseCypherParserListener{}

func (s *BaseCypherParserListener) VisitTerminal(node antlr.TerminalNode) {}

func (s *BaseCypherParserListener) VisitErrorNode(node antlr.ErrorNode) {}

func (s *BaseCypherParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

func (s *BaseCypherParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

func (s *BaseCypherParserListener) EnterScript(ctx *ScriptContext) {}

func (s *BaseCypherParserListener) ExitScript(ctx *ScriptContext) {}

func (s *BaseCypherParserListener) EnterQuery(ctx *QueryContext) {}

func (s *BaseCypherParserListener) ExitQuery(ctx *QueryContext) {}

func (s *BaseCypherParserListener) EnterRegularQuery(ctx *RegularQueryContext) {}

func (s *BaseCypherParserListener) ExitRegularQuery(ctx *RegularQueryContext) {}

func (s *BaseCypherParserListener) EnterSingleQuery(ctx *SingleQueryContext) {}

func (s *BaseCypherParserListener) ExitSingleQuery(ctx *SingleQueryContext) {}

func (s *BaseCypherParserListener) EnterStandaloneCall(ctx *StandaloneCallContext) {}

func (s *BaseCypherParserListener) ExitStandaloneCall(ctx *StandaloneCallContext) {}

func (s *BaseCypherParserListener) EnterReturnSt(ctx *ReturnStContext) {}

func (s *BaseCypherParserListener) ExitReturnSt(ctx *ReturnStContext) {}

func (s *BaseCypherParserListener) EnterWithSt(ctx *WithStContext) {}

func (s *BaseCypherParserListener) ExitWithSt(ctx *WithStContext) {}

func (s *BaseCypherParserListener) EnterSkipSt(ctx *SkipStContext) {}

func (s *BaseCypherParserListener) ExitSkipSt(ctx *SkipStContext) {}

func (s *BaseCypherParserListener) EnterLimitSt(ctx *LimitStContext) {}

func (s *BaseCypherParserListener) ExitLimitSt(ctx *LimitStContext) {}

func (s *BaseCypherParserListener) EnterProjectionBody(ctx *ProjectionBodyContext) {}

func (s *BaseCypherParserListener) ExitProjectionBody(ctx *ProjectionBodyContext) {}

func (s *BaseCypherParserListener) EnterProjectionItems(ctx *ProjectionItemsContext) {}

func (s *BaseCypherParserListener) ExitProjectionItems(ctx *ProjectionItemsContext) {}

func (s *BaseCypherParserListener) EnterProjectionItem(ctx *ProjectionItemContext) {}

func (s *BaseCypherParserListener) ExitProjectionItem(ctx *ProjectionItemContext) {}

func (s *BaseCypherParserListener) EnterOrderItem(ctx *OrderItemContext) {}

func (s *BaseCypherParserListener) ExitOrderItem(ctx *OrderItemContext) {}

func (s *BaseCypherParserListener) EnterOrderSt(ctx *OrderStContext) {}

func (s *BaseCypherParserListener) ExitOrderSt(ctx *OrderStContext) {}

func (s *BaseCypherParserListener) EnterSinglePartQ(ctx *SinglePartQContext) {}

func (s *BaseCypherParserListener) ExitSinglePartQ(ctx *SinglePartQContext) {}

func (s *BaseCypherParserListener) EnterMultiPartQ(ctx *MultiPartQContext) {}

func (s *BaseCypherParserListener) ExitMultiPartQ(ctx *MultiPartQContext) {}

func (s *BaseCypherParserListener) EnterMatchSt(ctx *MatchStContext) {}

func (s *BaseCypherParserListener) ExitMatchSt(ctx *MatchStContext) {}

func (s *BaseCypherParserListener) EnterUnwindSt(ctx *UnwindStContext) {}

func (s *BaseCypherParserListener) ExitUnwindSt(ctx *UnwindStContext) {}

func (s *BaseCypherParserListener) EnterReadingStatement(ctx *ReadingStatementContext) {}

func (s *BaseCypherParserListener) ExitReadingStatement(ctx *ReadingStatementContext) {}

func (s *BaseCypherParserListener) EnterUpdatingStatement(ctx *UpdatingStatementContext) {}

func (s *BaseCypherParserListener) ExitUpdatingStatement(ctx *UpdatingStatementContext) {}

func (s *BaseCypherParserListener) EnterDeleteSt(ctx *DeleteStContext) {}

func (s *BaseCypherParserListener) ExitDeleteSt(ctx *DeleteStContext) {}

func (s *BaseCypherParserListener) EnterRemoveSt(ctx *RemoveStContext) {}

func (s *BaseCypherParserListener) ExitRemoveSt(ctx *RemoveStContext) {}

func (s *BaseCypherParserListener) EnterRemoveItem(ctx *RemoveItemContext) {}

func (s *BaseCypherParserListener) ExitRemoveItem(ctx *RemoveItemContext) {}

func (s *BaseCypherParserListener) EnterQueryCallSt(ctx *QueryCallStContext) {}

func (s *BaseCypherParserListener) ExitQueryCallSt(ctx *QueryCallStContext) {}

func (s *BaseCypherParserListener) EnterParenExpressionChain(ctx *ParenExpressionChainContext) {}

func (s *BaseCypherParserListener) ExitParenExpressionChain(ctx *ParenExpressionChainContext) {}

func (s *BaseCypherParserListener) EnterYieldItems(ctx *YieldItemsContext) {}

func (s *BaseCypherParserListener) ExitYieldItems(ctx *YieldItemsContext) {}

func (s *BaseCypherParserListener) EnterYieldItem(ctx *YieldItemContext) {}

func (s *BaseCypherParserListener) ExitYieldItem(ctx *YieldItemContext) {}

func (s *BaseCypherParserListener) EnterMergeSt(ctx *MergeStContext) {}

func (s *BaseCypherParserListener) ExitMergeSt(ctx *MergeStContext) {}

func (s *BaseCypherParserListener) EnterMergeAction(ctx *MergeActionContext) {}

func (s *BaseCypherParserListener) ExitMergeAction(ctx *MergeActionContext) {}

func (s *BaseCypherParserListener) EnterSetSt(ctx *SetStContext) {}

func (s *BaseCypherParserListener) ExitSetSt(ctx *SetStContext) {}

func (s *BaseCypherParserListener) EnterSetItem(ctx *SetItemContext) {}

func (s *BaseCypherParserListener) ExitSetItem(ctx *SetItemContext) {}

func (s *BaseCypherParserListener) EnterNodeLabels(ctx *NodeLabelsContext) {}

func (s *BaseCypherParserListener) ExitNodeLabels(ctx *NodeLabelsContext) {}

func (s *BaseCypherParserListener) EnterCreateSt(ctx *CreateStContext) {}

func (s *BaseCypherParserListener) ExitCreateSt(ctx *CreateStContext) {}

func (s *BaseCypherParserListener) EnterPatternWhere(ctx *PatternWhereContext) {}

func (s *BaseCypherParserListener) ExitPatternWhere(ctx *PatternWhereContext) {}

func (s *BaseCypherParserListener) EnterWhere(ctx *WhereContext) {}

func (s *BaseCypherParserListener) ExitWhere(ctx *WhereContext) {}

func (s *BaseCypherParserListener) EnterPattern(ctx *PatternContext) {}

func (s *BaseCypherParserListener) ExitPattern(ctx *PatternContext) {}

func (s *BaseCypherParserListener) EnterExpression(ctx *ExpressionContext) {}

func (s *BaseCypherParserListener) ExitExpression(ctx *ExpressionContext) {}

func (s *BaseCypherParserListener) EnterXorExpression(ctx *XorExpressionContext) {}

func (s *BaseCypherParserListener) ExitXorExpression(ctx *XorExpressionContext) {}

func (s *BaseCypherParserListener) EnterAndExpression(ctx *AndExpressionContext) {}

func (s *BaseCypherParserListener) ExitAndExpression(ctx *AndExpressionContext) {}

func (s *BaseCypherParserListener) EnterNotExpression(ctx *NotExpressionContext) {}

func (s *BaseCypherParserListener) ExitNotExpression(ctx *NotExpressionContext) {}

func (s *BaseCypherParserListener) EnterComparisonExpression(ctx *ComparisonExpressionContext) {}

func (s *BaseCypherParserListener) ExitComparisonExpression(ctx *ComparisonExpressionContext) {}

func (s *BaseCypherParserListener) EnterComparisonSigns(ctx *ComparisonSignsContext) {}

func (s *BaseCypherParserListener) ExitComparisonSigns(ctx *ComparisonSignsContext) {}

func (s *BaseCypherParserListener) EnterAddSubExpression(ctx *AddSubExpressionContext) {}

func (s *BaseCypherParserListener) ExitAddSubExpression(ctx *AddSubExpressionContext) {}

func (s *BaseCypherParserListener) EnterMultDivExpression(ctx *MultDivExpressionContext) {}

func (s *BaseCypherParserListener) ExitMultDivExpression(ctx *MultDivExpressionContext) {}

func (s *BaseCypherParserListener) EnterPowerExpression(ctx *PowerExpressionContext) {}

func (s *BaseCypherParserListener) ExitPowerExpression(ctx *PowerExpressionContext) {}

func (s *BaseCypherParserListener) EnterUnaryAddSubExpression(ctx *UnaryAddSubExpressionContext) {}

func (s *BaseCypherParserListener) ExitUnaryAddSubExpression(ctx *UnaryAddSubExpressionContext) {}

func (s *BaseCypherParserListener) EnterAtomicExpression(ctx *AtomicExpressionContext) {}

func (s *BaseCypherParserListener) ExitAtomicExpression(ctx *AtomicExpressionContext) {}

func (s *BaseCypherParserListener) EnterListExpression(ctx *ListExpressionContext) {}

func (s *BaseCypherParserListener) ExitListExpression(ctx *ListExpressionContext) {}

func (s *BaseCypherParserListener) EnterStringExpression(ctx *StringExpressionContext) {}

func (s *BaseCypherParserListener) ExitStringExpression(ctx *StringExpressionContext) {}

func (s *BaseCypherParserListener) EnterStringExpPrefix(ctx *StringExpPrefixContext) {}

func (s *BaseCypherParserListener) ExitStringExpPrefix(ctx *StringExpPrefixContext) {}

func (s *BaseCypherParserListener) EnterNullExpression(ctx *NullExpressionContext) {}

func (s *BaseCypherParserListener) ExitNullExpression(ctx *NullExpressionContext) {}

func (s *BaseCypherParserListener) EnterPropertyOrLabelExpression(ctx *PropertyOrLabelExpressionContext) {
}

func (s *BaseCypherParserListener) ExitPropertyOrLabelExpression(ctx *PropertyOrLabelExpressionContext) {
}

func (s *BaseCypherParserListener) EnterPropertyExpression(ctx *PropertyExpressionContext) {}

func (s *BaseCypherParserListener) ExitPropertyExpression(ctx *PropertyExpressionContext) {}

func (s *BaseCypherParserListener) EnterPatternPart(ctx *PatternPartContext) {}

func (s *BaseCypherParserListener) ExitPatternPart(ctx *PatternPartContext) {}

func (s *BaseCypherParserListener) EnterPatternElem(ctx *PatternElemContext) {}

func (s *BaseCypherParserListener) ExitPatternElem(ctx *PatternElemContext) {}

func (s *BaseCypherParserListener) EnterPatternElemChain(ctx *PatternElemChainContext) {}

func (s *BaseCypherParserListener) ExitPatternElemChain(ctx *PatternElemChainContext) {}

func (s *BaseCypherParserListener) EnterProperties(ctx *PropertiesContext) {}

func (s *BaseCypherParserListener) ExitProperties(ctx *PropertiesContext) {}

func (s *BaseCypherParserListener) EnterNodePattern(ctx *NodePatternContext) {}

func (s *BaseCypherParserListener) ExitNodePattern(ctx *NodePatternContext) {}

func (s *BaseCypherParserListener) EnterAtom(ctx *AtomContext) {}

func (s *BaseCypherParserListener) ExitAtom(ctx *AtomContext) {}

func (s *BaseCypherParserListener) EnterLhs(ctx *LhsContext) {}

func (s *BaseCypherParserListener) ExitLhs(ctx *LhsContext) {}

func (s *BaseCypherParserListener) EnterRelationshipPattern(ctx *RelationshipPatternContext) {}

func (s *BaseCypherParserListener) ExitRelationshipPattern(ctx *RelationshipPatternContext) {}

func (s *BaseCypherParserListener) EnterRelationDetail(ctx *RelationDetailContext) {}

func (s *BaseCypherParserListener) ExitRelationDetail(ctx *RelationDetailContext) {}

func (s *BaseCypherParserListener) EnterRelationshipTypes(ctx *RelationshipTypesContext) {}

func (s *BaseCypherParserListener) ExitRelationshipTypes(ctx *RelationshipTypesContext) {}

func (s *BaseCypherParserListener) EnterUnionSt(ctx *UnionStContext) {}

func (s *BaseCypherParserListener) ExitUnionSt(ctx *UnionStContext) {}

func (s *BaseCypherParserListener) EnterSubqueryExist(ctx *SubqueryExistContext) {}

func (s *BaseCypherParserListener) ExitSubqueryExist(ctx *SubqueryExistContext) {}

func (s *BaseCypherParserListener) EnterInvocationName(ctx *InvocationNameContext) {}

func (s *BaseCypherParserListener) ExitInvocationName(ctx *InvocationNameContext) {}

func (s *BaseCypherParserListener) EnterFunctionInvocation(ctx *FunctionInvocationContext) {}

func (s *BaseCypherParserListener) ExitFunctionInvocation(ctx *FunctionInvocationContext) {}

func (s *BaseCypherParserListener) EnterParenthesizedExpression(ctx *ParenthesizedExpressionContext) {
}

func (s *BaseCypherParserListener) ExitParenthesizedExpression(ctx *ParenthesizedExpressionContext) {}

func (s *BaseCypherParserListener) EnterFilterWith(ctx *FilterWithContext) {}

func (s *BaseCypherParserListener) ExitFilterWith(ctx *FilterWithContext) {}

func (s *BaseCypherParserListener) EnterPatternComprehension(ctx *PatternComprehensionContext) {}

func (s *BaseCypherParserListener) ExitPatternComprehension(ctx *PatternComprehensionContext) {}

func (s *BaseCypherParserListener) EnterRelationshipsChainPattern(ctx *RelationshipsChainPatternContext) {
}

func (s *BaseCypherParserListener) ExitRelationshipsChainPattern(ctx *RelationshipsChainPatternContext) {
}

func (s *BaseCypherParserListener) EnterListComprehension(ctx *ListComprehensionContext) {}

func (s *BaseCypherParserListener) ExitListComprehension(ctx *ListComprehensionContext) {}

func (s *BaseCypherParserListener) EnterFilterExpression(ctx *FilterExpressionContext) {}

func (s *BaseCypherParserListener) ExitFilterExpression(ctx *FilterExpressionContext) {}

func (s *BaseCypherParserListener) EnterCountAll(ctx *CountAllContext) {}

func (s *BaseCypherParserListener) ExitCountAll(ctx *CountAllContext) {}

func (s *BaseCypherParserListener) EnterExpressionChain(ctx *ExpressionChainContext) {}

func (s *BaseCypherParserListener) ExitExpressionChain(ctx *ExpressionChainContext) {}

func (s *BaseCypherParserListener) EnterCaseExpression(ctx *CaseExpressionContext) {}

func (s *BaseCypherParserListener) ExitCaseExpression(ctx *CaseExpressionContext) {}

func (s *BaseCypherParserListener) EnterParameter(ctx *ParameterContext) {}

func (s *BaseCypherParserListener) ExitParameter(ctx *ParameterContext) {}

func (s *BaseCypherParserListener) EnterLiteral(ctx *LiteralContext) {}

func (s *BaseCypherParserListener) ExitLiteral(ctx *LiteralContext) {}

func (s *BaseCypherParserListener) EnterRangeLit(ctx *RangeLitContext) {}

func (s *BaseCypherParserListener) ExitRangeLit(ctx *RangeLitContext) {}

func (s *BaseCypherParserListener) EnterBoolLit(ctx *BoolLitContext) {}

func (s *BaseCypherParserListener) ExitBoolLit(ctx *BoolLitContext) {}

func (s *BaseCypherParserListener) EnterNumLit(ctx *NumLitContext) {}

func (s *BaseCypherParserListener) ExitNumLit(ctx *NumLitContext) {}

func (s *BaseCypherParserListener) EnterStringLit(ctx *StringLitContext) {}

func (s *BaseCypherParserListener) ExitStringLit(ctx *StringLitContext) {}

func (s *BaseCypherParserListener) EnterCharLit(ctx *CharLitContext) {}

func (s *BaseCypherParserListener) ExitCharLit(ctx *CharLitContext) {}

func (s *BaseCypherParserListener) EnterListLit(ctx *ListLitContext) {}

func (s *BaseCypherParserListener) ExitListLit(ctx *ListLitContext) {}

func (s *BaseCypherParserListener) EnterMapLit(ctx *MapLitContext) {}

func (s *BaseCypherParserListener) ExitMapLit(ctx *MapLitContext) {}

func (s *BaseCypherParserListener) EnterMapPair(ctx *MapPairContext) {}

func (s *BaseCypherParserListener) ExitMapPair(ctx *MapPairContext) {}

func (s *BaseCypherParserListener) EnterName(ctx *NameContext) {}

func (s *BaseCypherParserListener) ExitName(ctx *NameContext) {}

func (s *BaseCypherParserListener) EnterSymbol(ctx *SymbolContext) {}

func (s *BaseCypherParserListener) ExitSymbol(ctx *SymbolContext) {}

func (s *BaseCypherParserListener) EnterReservedWord(ctx *ReservedWordContext) {}

func (s *BaseCypherParserListener) ExitReservedWord(ctx *ReservedWordContext) {}
