-- Functional rollback should retain these tables. This inverse is only for
-- an unused installation, and refuses to discard any decisions or receipts.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM classification_state) OR EXISTS(SELECT 1 FROM classification_events)
 OR EXISTS(SELECT 1 FROM classification_tag_decisions) OR EXISTS(SELECT 1 FROM classification_rejections)
 OR EXISTS(SELECT 1 FROM classification_operations) OR EXISTS(SELECT 1 FROM classification_rule_versions)
 OR EXISTS(SELECT 1 FROM transaction_suggestions WHERE identity<>'' OR context_key<>'' OR base_revision<>0)
 THEN RAISE EXCEPTION 'Classification decisions exist; rollback refused'; END IF;
END $$;
DROP TRIGGER classification_domain_delete ON domains;
DROP TRIGGER classification_link_update ON transaction_tags;
DROP TRIGGER classification_events_immutable ON classification_events;
DROP TRIGGER classification_rules ON rules;
DROP TRIGGER classification_dictionary ON tags;
DROP TRIGGER classification_tag_after ON transaction_tags;
DROP TRIGGER classification_tag_before ON transaction_tags;
DROP TRIGGER classification_transaction_update ON transactions;
DROP TRIGGER classification_transaction_insert ON transactions;
DROP FUNCTION classification_domain_delete_guard();
DROP FUNCTION classification_link_update_refused();
DROP FUNCTION classification_event_immutable();
DROP FUNCTION classification_rule_version();
DROP FUNCTION classification_dictionary_guard();
DROP FUNCTION classification_tag_guard();
DROP FUNCTION classification_transaction_guard();
DROP FUNCTION classification_tag_decide(bigint,bigint,bigint,text);
DROP FUNCTION classification_set_locks(bigint,bigint,boolean,boolean);
DROP FUNCTION classification_finish(bigint,bigint);
DROP FUNCTION classification_begin(bigint,bigint,text,text,text,bigint);
DROP FUNCTION classification_record(bigint,bigint,text,jsonb,text,bigint,boolean);
DROP FUNCTION classification_snapshot(bigint,bigint);
DROP TABLE classification_operations;
DROP TABLE classification_events;
DROP TABLE classification_rejections;
DROP TABLE classification_tag_decisions;
DROP TABLE classification_rule_versions;
DROP TABLE classification_state;
ALTER TABLE transaction_suggestions DROP COLUMN base_revision;
ALTER TABLE transaction_suggestions DROP COLUMN context_key;
ALTER TABLE transaction_suggestions DROP COLUMN identity;
DELETE FROM schema_migrations WHERE name='035_classification_decisions.sql';
