package cypher

import "container/antlr"

type CypherParserListener interface {
	antlr.ParseTreeListener

	EnterScript(c *ScriptContext)

	EnterQuery(c *QueryContext)

	EnterRegularQuery(c *RegularQueryContext)

	EnterSingleQuery(c *SingleQueryContext)

	EnterStandaloneCall(c *StandaloneCallContext)

	EnterReturnSt(c *ReturnStContext)

	EnterWithSt(c *WithStContext)

	EnterSkipSt(c *SkipStContext)

	EnterLimitSt(c *LimitStContext)

	EnterProjectionBody(c *ProjectionBodyContext)

	EnterProjectionItems(c *ProjectionItemsContext)

	EnterProjectionItem(c *ProjectionItemContext)

	EnterOrderItem(c *OrderItemContext)

	EnterOrderSt(c *OrderStContext)

	EnterSinglePartQ(c *SinglePartQContext)

	EnterMultiPartQ(c *MultiPartQContext)

	EnterMatchSt(c *MatchStContext)

	EnterUnwindSt(c *UnwindStContext)

	EnterReadingStatement(c *ReadingStatementContext)

	EnterUpdatingStatement(c *UpdatingStatementContext)

	EnterDeleteSt(c *DeleteStContext)

	EnterRemoveSt(c *RemoveStContext)

	EnterRemoveItem(c *RemoveItemContext)

	EnterQueryCallSt(c *QueryCallStContext)

	EnterParenExpressionChain(c *ParenExpressionChainContext)

	EnterYieldItems(c *YieldItemsContext)

	EnterYieldItem(c *YieldItemContext)

	EnterMergeSt(c *MergeStContext)

	EnterMergeAction(c *MergeActionContext)

	EnterSetSt(c *SetStContext)

	EnterSetItem(c *SetItemContext)

	EnterNodeLabels(c *NodeLabelsContext)

	EnterCreateSt(c *CreateStContext)

	EnterPatternWhere(c *PatternWhereContext)

	EnterWhere(c *WhereContext)

	EnterPattern(c *PatternContext)

	EnterExpression(c *ExpressionContext)

	EnterXorExpression(c *XorExpressionContext)

	EnterAndExpression(c *AndExpressionContext)

	EnterNotExpression(c *NotExpressionContext)

	EnterComparisonExpression(c *ComparisonExpressionContext)

	EnterComparisonSigns(c *ComparisonSignsContext)

	EnterAddSubExpression(c *AddSubExpressionContext)

	EnterMultDivExpression(c *MultDivExpressionContext)

	EnterPowerExpression(c *PowerExpressionContext)

	EnterUnaryAddSubExpression(c *UnaryAddSubExpressionContext)

	EnterAtomicExpression(c *AtomicExpressionContext)

	EnterListExpression(c *ListExpressionContext)

	EnterStringExpression(c *StringExpressionContext)

	EnterStringExpPrefix(c *StringExpPrefixContext)

	EnterNullExpression(c *NullExpressionContext)

	EnterPropertyOrLabelExpression(c *PropertyOrLabelExpressionContext)

	EnterPropertyExpression(c *PropertyExpressionContext)

	EnterPatternPart(c *PatternPartContext)

	EnterPatternElem(c *PatternElemContext)

	EnterPatternElemChain(c *PatternElemChainContext)

	EnterProperties(c *PropertiesContext)

	EnterNodePattern(c *NodePatternContext)

	EnterAtom(c *AtomContext)

	EnterLhs(c *LhsContext)

	EnterRelationshipPattern(c *RelationshipPatternContext)

	EnterRelationDetail(c *RelationDetailContext)

	EnterRelationshipTypes(c *RelationshipTypesContext)

	EnterUnionSt(c *UnionStContext)

	EnterSubqueryExist(c *SubqueryExistContext)

	EnterInvocationName(c *InvocationNameContext)

	EnterFunctionInvocation(c *FunctionInvocationContext)

	EnterParenthesizedExpression(c *ParenthesizedExpressionContext)

	EnterFilterWith(c *FilterWithContext)

	EnterPatternComprehension(c *PatternComprehensionContext)

	EnterRelationshipsChainPattern(c *RelationshipsChainPatternContext)

	EnterListComprehension(c *ListComprehensionContext)

	EnterFilterExpression(c *FilterExpressionContext)

	EnterCountAll(c *CountAllContext)

	EnterExpressionChain(c *ExpressionChainContext)

	EnterCaseExpression(c *CaseExpressionContext)

	EnterParameter(c *ParameterContext)

	EnterLiteral(c *LiteralContext)

	EnterRangeLit(c *RangeLitContext)

	EnterBoolLit(c *BoolLitContext)

	EnterNumLit(c *NumLitContext)

	EnterStringLit(c *StringLitContext)

	EnterCharLit(c *CharLitContext)

	EnterListLit(c *ListLitContext)

	EnterMapLit(c *MapLitContext)

	EnterMapPair(c *MapPairContext)

	EnterName(c *NameContext)

	EnterSymbol(c *SymbolContext)

	EnterReservedWord(c *ReservedWordContext)

	ExitScript(c *ScriptContext)

	ExitQuery(c *QueryContext)

	ExitRegularQuery(c *RegularQueryContext)

	ExitSingleQuery(c *SingleQueryContext)

	ExitStandaloneCall(c *StandaloneCallContext)

	ExitReturnSt(c *ReturnStContext)

	ExitWithSt(c *WithStContext)

	ExitSkipSt(c *SkipStContext)

	ExitLimitSt(c *LimitStContext)

	ExitProjectionBody(c *ProjectionBodyContext)

	ExitProjectionItems(c *ProjectionItemsContext)

	ExitProjectionItem(c *ProjectionItemContext)

	ExitOrderItem(c *OrderItemContext)

	ExitOrderSt(c *OrderStContext)

	ExitSinglePartQ(c *SinglePartQContext)

	ExitMultiPartQ(c *MultiPartQContext)

	ExitMatchSt(c *MatchStContext)

	ExitUnwindSt(c *UnwindStContext)

	ExitReadingStatement(c *ReadingStatementContext)

	ExitUpdatingStatement(c *UpdatingStatementContext)

	ExitDeleteSt(c *DeleteStContext)

	ExitRemoveSt(c *RemoveStContext)

	ExitRemoveItem(c *RemoveItemContext)

	ExitQueryCallSt(c *QueryCallStContext)

	ExitParenExpressionChain(c *ParenExpressionChainContext)

	ExitYieldItems(c *YieldItemsContext)

	ExitYieldItem(c *YieldItemContext)

	ExitMergeSt(c *MergeStContext)

	ExitMergeAction(c *MergeActionContext)

	ExitSetSt(c *SetStContext)

	ExitSetItem(c *SetItemContext)

	ExitNodeLabels(c *NodeLabelsContext)

	ExitCreateSt(c *CreateStContext)

	ExitPatternWhere(c *PatternWhereContext)

	ExitWhere(c *WhereContext)

	ExitPattern(c *PatternContext)

	ExitExpression(c *ExpressionContext)

	ExitXorExpression(c *XorExpressionContext)

	ExitAndExpression(c *AndExpressionContext)

	ExitNotExpression(c *NotExpressionContext)

	ExitComparisonExpression(c *ComparisonExpressionContext)

	ExitComparisonSigns(c *ComparisonSignsContext)

	ExitAddSubExpression(c *AddSubExpressionContext)

	ExitMultDivExpression(c *MultDivExpressionContext)

	ExitPowerExpression(c *PowerExpressionContext)

	ExitUnaryAddSubExpression(c *UnaryAddSubExpressionContext)

	ExitAtomicExpression(c *AtomicExpressionContext)

	ExitListExpression(c *ListExpressionContext)

	ExitStringExpression(c *StringExpressionContext)

	ExitStringExpPrefix(c *StringExpPrefixContext)

	ExitNullExpression(c *NullExpressionContext)

	ExitPropertyOrLabelExpression(c *PropertyOrLabelExpressionContext)

	ExitPropertyExpression(c *PropertyExpressionContext)

	ExitPatternPart(c *PatternPartContext)

	ExitPatternElem(c *PatternElemContext)

	ExitPatternElemChain(c *PatternElemChainContext)

	ExitProperties(c *PropertiesContext)

	ExitNodePattern(c *NodePatternContext)

	ExitAtom(c *AtomContext)

	ExitLhs(c *LhsContext)

	ExitRelationshipPattern(c *RelationshipPatternContext)

	ExitRelationDetail(c *RelationDetailContext)

	ExitRelationshipTypes(c *RelationshipTypesContext)

	ExitUnionSt(c *UnionStContext)

	ExitSubqueryExist(c *SubqueryExistContext)

	ExitInvocationName(c *InvocationNameContext)

	ExitFunctionInvocation(c *FunctionInvocationContext)

	ExitParenthesizedExpression(c *ParenthesizedExpressionContext)

	ExitFilterWith(c *FilterWithContext)

	ExitPatternComprehension(c *PatternComprehensionContext)

	ExitRelationshipsChainPattern(c *RelationshipsChainPatternContext)

	ExitListComprehension(c *ListComprehensionContext)

	ExitFilterExpression(c *FilterExpressionContext)

	ExitCountAll(c *CountAllContext)

	ExitExpressionChain(c *ExpressionChainContext)

	ExitCaseExpression(c *CaseExpressionContext)

	ExitParameter(c *ParameterContext)

	ExitLiteral(c *LiteralContext)

	ExitRangeLit(c *RangeLitContext)

	ExitBoolLit(c *BoolLitContext)

	ExitNumLit(c *NumLitContext)

	ExitStringLit(c *StringLitContext)

	ExitCharLit(c *CharLitContext)

	ExitListLit(c *ListLitContext)

	ExitMapLit(c *MapLitContext)

	ExitMapPair(c *MapPairContext)

	ExitName(c *NameContext)

	ExitSymbol(c *SymbolContext)

	ExitReservedWord(c *ReservedWordContext)
}
