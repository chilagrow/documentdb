SET search_path TO helio_core,helio_api,helio_api_catalog,helio_api_internal;
SET citus.next_shard_id TO 884500;
SET helio_api.next_collection_id TO 88450;
SET helio_api.next_collection_index_id TO 88450;

CREATE SCHEMA setWindowFieldSchema;

-- The test is intended to set an agreement between the setwindowfields and group aggregate operators
-- to be working on both cases (if implemented)
-- [Important] - When any operator is supported for both $group and $setWindowFields then please remove it from here.
CREATE OR REPLACE FUNCTION get_unsupported_operators()
RETURNS TABLE(unsupported_operator text) AS $$
BEGIN
    RETURN QUERY SELECT unnest(ARRAY[
        '$bottom',
        '$bottomN',
        '$maxN',
        '$median',
        '$minN',
        '$percentile',
        '$stdDevSamp',
        '$stdDevPop',
        '$top',
        '$topN'
    ]) AS unsupported_operator;
END;
$$ LANGUAGE plpgsql;


-- Make a non-empty collection
SELECT helio_api.insert_one('db','setWindowField_compliance','{ "_id": 1 }', NULL);

DO $$
DECLARE
    operator text;
    errorMessage text;
    supportedInSetWindowFields BOOLEAN;
    supportedInGroup BOOLEAN;
    querySpec bson;
BEGIN
    FOR operator IN SELECT * FROM get_unsupported_operators()
    LOOP
        supportedInSetWindowFields := TRUE;
        supportedInGroup := TRUE;
        querySpec := FORMAT('{ "aggregate": "setWindowField_compliance", "pipeline":  [{"$group": { "_id": "$_id", "test": {"%s": { } } }}]}', operator)::helio_core.bson;
        BEGIN
            SELECT document FROM helio_api_catalog.bson_aggregation_pipeline('db', querySpec);
        EXCEPTION WHEN OTHERS THEN
            errorMessage := SQLERRM;
            IF errorMessage LIKE '%Unknown group operator%' OR errorMessage LIKE '%not implemented%' THEN
                supportedInGroup := FALSE;
            END IF;
        END;

        querySpec := FORMAT('{ "aggregate": "setWindowField_compliance", "pipeline":  [{"$setWindowFields": { "output": { "field": { "%s": { } } } }}]}', operator)::helio_core.bson;
        BEGIN
            SELECT document FROM helio_api_catalog.bson_aggregation_pipeline('db', querySpec);
        EXCEPTION WHEN OTHERS THEN
            errorMessage := SQLERRM;
            IF errorMessage LIKE '%not supported%' THEN
                supportedInSetWindowFields := FALSE;
            END IF;
        END;

        IF supportedInSetWindowFields <> supportedInGroup THEN
            RAISE NOTICE '[TEST FAILED] Operator % is not supported in %', operator, CASE WHEN supportedInSetWindowFields THEN '$group' ELSE '$setWindowFields' END;
        ELSEIF supportedInSetWindowFields THEN
            RAISE NOTICE '[TEST PASSED] Operator % is supported in both $group and $setWindowFields', operator;
        ELSE
            RAISE NOTICE '[TEST PASSED] Operator % is not supported in both $group and $setWindowFields', operator;
        END IF;
    END LOOP;
END $$;

DROP SCHEMA setWindowFieldSchema CASCADE;
