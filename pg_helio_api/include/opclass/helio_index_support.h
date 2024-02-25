/*-------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation.  All rights reserved.
 *
 * include/opclass/helio_index_support.h
 *
 * Common declarations for Index support functions.
 *
 *-------------------------------------------------------------------------
 */

#ifndef HELIO_INDEX_SUPPORT_H
#define HELIO_INDEX_SUPPORT_H

#include <opclass/helio_bson_text_gin.h>
#include <vector/vector_utilities.h>
#include <optimizer/planner.h>
struct IndexOptInfo;

/*
 * Input immutable data for the ReplaceExtensionFunctionContext
 */
typedef struct ReplaceFunctionContextInput
{
	/* Whether or not to do a runtime check for $text */
	bool isRuntimeTextScan;

	/* Whether or not this is the query on the actual Shard table */
	bool isShardQuery;

	/* CollectionId of the base collection if it's known */
	uint64 collectionId;
} ReplaceFunctionContextInput;

/*
 * Context object passed between ReplaceExtensionFunctionOperatorsInPaths
 * and ReplaceExtensionFunctionOperatorsInRestrictionPaths. This takes context
 * about what index paths were replaced and uses that in the replacement of
 * restriction paths.
 */
typedef struct ReplaceExtensionFunctionContext
{
	/* The query data used for Text indexes (can have NULL indexOptions) */
	QueryTextIndexData indexOptionsForText;
	SearchQueryEvalData queryDataForVectorSearch;

	/* Whether or not the index paths/restriction paths have text query */
	bool hasTextIndexQuery;

	/* Whether or not the index paths/restriction paths have vector search query */
	bool hasVectorSearchQuery;

	/* Whether or not the index paths already has a primary key lookup */
	IndexPath *primaryKeyLookupPath;

	/* The input data context for the call */
	ReplaceFunctionContextInput inputData;
} ReplaceExtensionFunctionContext;

/* Type of the parent node in the query plan of a query for $in optimization. This is not
 * intended for general use */
typedef enum PlanParentType
{
	/* Don't perform $in rewrite when parent is invalid */
	PARENTTYPE_INVALID = 0,

	/* Perform rewrite, but the rewritten BitmapORPath needs to be wrapped in a BitMapHeapPath*/
	PARENTTYPE_NONE,

	/* Peform rewrite into a BitmapORPath*/
	PARENTTYPE_BITMAPHEAP
}PlanParentType;

extern bool EnableInQueryOptimization;

List * ReplaceExtensionFunctionOperatorsInRestrictionPaths(List *restrictInfo,
														   ReplaceExtensionFunctionContext
														   *context);
void ReplaceExtensionFunctionOperatorsInPaths(PlannerInfo *root, RelOptInfo *rel,
											  List *pathsList, PlanParentType parentType,
											  ReplaceExtensionFunctionContext *context);


bool IsBtreePrimaryKeyIndex(struct IndexOptInfo *indexInfo);
#endif
