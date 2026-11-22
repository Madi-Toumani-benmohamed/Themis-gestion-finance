-- Additive metadata only: no UPDATE/backfill of transactions or classifications.
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_identity_owner ON tags(id,owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_rules_identity_owner ON rules(id,owner_id);

CREATE TABLE classification_state (
 owner_id BIGINT NOT NULL, transaction_id INTEGER NOT NULL,
 revision BIGINT NOT NULL DEFAULT 0 CHECK(revision>=0),
 context_generation BIGINT NOT NULL DEFAULT 0 CHECK(context_generation>=0),
 domain_locked BOOLEAN NOT NULL DEFAULT false, tags_locked BOOLEAN NOT NULL DEFAULT false,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_id,transaction_id),
 FOREIGN KEY(transaction_id,owner_id) REFERENCES transactions(id,owner_id) ON DELETE CASCADE
);
CREATE TABLE classification_tag_decisions (
 owner_id BIGINT NOT NULL, transaction_id INTEGER NOT NULL, tag_id BIGINT NOT NULL,
 decision TEXT NOT NULL CHECK(decision IN ('added','removed')),
 revision BIGINT NOT NULL, origin TEXT NOT NULL, decided_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_id,transaction_id,tag_id),
 FOREIGN KEY(transaction_id,owner_id) REFERENCES transactions(id,owner_id) ON DELETE CASCADE,
 FOREIGN KEY(tag_id,owner_id) REFERENCES tags(id,owner_id) ON DELETE RESTRICT
);
CREATE TABLE classification_rejections (
 owner_id BIGINT NOT NULL, transaction_id INTEGER NOT NULL,
 identity TEXT NOT NULL CHECK(length(identity)=64), context_key TEXT NOT NULL CHECK(length(context_key)=64),
 revision BIGINT NOT NULL, origin TEXT NOT NULL, rejected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_id,transaction_id,identity,context_key),
 FOREIGN KEY(transaction_id,owner_id) REFERENCES transactions(id,owner_id) ON DELETE CASCADE
);
CREATE TABLE classification_events (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, transaction_id INTEGER NOT NULL,
 revision BIGINT NOT NULL CHECK(revision>0), occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 origin TEXT NOT NULL, actor_id BIGINT,
 event_type TEXT NOT NULL, rule_id BIGINT, tag_id BIGINT,
 suggestion_identity TEXT CHECK(suggestion_identity IS NULL OR length(suggestion_identity)=64),
 before_state JSONB NOT NULL, after_state JSONB NOT NULL,
 reason TEXT NOT NULL CHECK(length(reason)<=160), operation_id TEXT NOT NULL CHECK(length(operation_id)<=100),
 UNIQUE(owner_id,transaction_id,revision),
 CHECK(octet_length(before_state::text)<=65536 AND octet_length(after_state::text)<=65536)
);
CREATE INDEX idx_classification_events_owner_tx ON classification_events(owner_id,transaction_id,id DESC);
CREATE TABLE classification_operations (
 owner_id BIGINT NOT NULL, transaction_id INTEGER NOT NULL, operation_id TEXT NOT NULL CHECK(length(operation_id) BETWEEN 1 AND 100),
 request_hash TEXT NOT NULL CHECK(length(request_hash)=64), revision BIGINT NOT NULL,
 completed_at TIMESTAMPTZ NOT NULL DEFAULT now(), PRIMARY KEY(owner_id,transaction_id,operation_id),
 FOREIGN KEY(transaction_id,owner_id) REFERENCES transactions(id,owner_id) ON DELETE CASCADE
);
CREATE TABLE classification_rule_versions (
 owner_id BIGINT NOT NULL, rule_id BIGINT NOT NULL, revision BIGINT NOT NULL DEFAULT 1,
 PRIMARY KEY(owner_id,rule_id), FOREIGN KEY(rule_id,owner_id) REFERENCES rules(id,owner_id) ON DELETE CASCADE
);
ALTER TABLE transaction_suggestions ADD COLUMN identity TEXT NOT NULL DEFAULT '';
ALTER TABLE transaction_suggestions ADD COLUMN context_key TEXT NOT NULL DEFAULT '';
ALTER TABLE transaction_suggestions ADD COLUMN base_revision BIGINT NOT NULL DEFAULT 0;

CREATE FUNCTION classification_snapshot(own BIGINT, tid BIGINT) RETURNS JSONB LANGUAGE sql AS $$
 SELECT jsonb_build_object('domain_id',t.domain_id,'source',t.classification_source,
 'tag_ids',COALESCE((SELECT jsonb_agg(tt.tag_id ORDER BY tt.tag_id) FROM transaction_tags tt JOIN tags g ON g.id=tt.tag_id AND g.owner_id=own WHERE tt.transaction_id=tid),'[]'::jsonb),
 'tag_decisions',COALESCE((SELECT jsonb_agg(jsonb_build_object('tag_id',x.tag_id,'decision',x.decision) ORDER BY x.tag_id) FROM classification_tag_decisions x WHERE x.owner_id=own AND x.transaction_id=tid),'[]'::jsonb),
 'revision',COALESCE(s.revision,0),'context_generation',COALESCE(s.context_generation,0),
 'domain_locked',COALESCE(s.domain_locked,false),'tags_locked',COALESCE(s.tags_locked,false))
 FROM transactions t LEFT JOIN classification_state s ON s.owner_id=t.owner_id AND s.transaction_id=t.id
 WHERE t.owner_id=own AND t.id=tid
$$;

CREATE FUNCTION classification_record(own BIGINT,tid BIGINT,kind TEXT,previous JSONB,reason_code TEXT,tag BIGINT DEFAULT NULL,bump_context BOOLEAN DEFAULT true)
RETURNS BIGINT LANGUAGE plpgsql AS $$
DECLARE rev BIGINT; org TEXT; actor BIGINT; op TEXT; rid BIGINT;
BEGIN
 PERFORM id FROM transactions WHERE id=tid AND owner_id=own FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'classification transaction unavailable' USING ERRCODE='P0002'; END IF;
 org:=COALESCE(NULLIF(current_setting('themis.classification_origin',true),''),'legacy_sql');
 actor:=NULLIF(current_setting('themis.classification_actor',true),'')::bigint;
 op:=COALESCE(NULLIF(current_setting('themis.classification_operation',true),''),gen_random_uuid()::text);
 rid:=NULLIF(current_setting('themis.classification_rule',true),'')::bigint;
 IF rid IS NOT NULL AND NOT EXISTS(SELECT 1 FROM rules WHERE id=rid AND owner_id=own) THEN RAISE EXCEPTION 'classification rule unavailable'; END IF;
 INSERT INTO classification_state(owner_id,transaction_id) VALUES(own,tid) ON CONFLICT DO NOTHING;
 UPDATE classification_state SET revision=revision+1,context_generation=context_generation+CASE WHEN bump_context THEN 1 ELSE 0 END,updated_at=now()
 WHERE owner_id=own AND transaction_id=tid RETURNING revision INTO rev;
 INSERT INTO classification_events(owner_id,transaction_id,revision,origin,actor_id,event_type,rule_id,tag_id,before_state,after_state,reason,operation_id,suggestion_identity)
 VALUES(own,tid,rev,org,actor,kind,rid,tag,previous,classification_snapshot(own,tid),reason_code,op,NULLIF(current_setting('themis.classification_suggestion',true),''));
 RETURN rev;
END $$;

CREATE FUNCTION classification_begin(own BIGINT,tid BIGINT,org TEXT,op TEXT,req TEXT,expected BIGINT DEFAULT NULL)
RETURNS BOOLEAN LANGUAGE plpgsql AS $$
DECLARE old_req TEXT; rev BIGINT;
BEGIN
 PERFORM id FROM transactions WHERE id=tid AND owner_id=own FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'classification transaction unavailable' USING ERRCODE='P0002'; END IF;
 SELECT request_hash INTO old_req FROM classification_operations WHERE owner_id=own AND transaction_id=tid AND operation_id=op;
 IF FOUND THEN
  IF old_req<>req THEN RAISE EXCEPTION 'classification operation reused' USING ERRCODE='40001'; END IF;
  RETURN true;
 END IF;
 SELECT COALESCE((SELECT revision FROM classification_state WHERE owner_id=own AND transaction_id=tid),0) INTO rev;
 IF expected IS NOT NULL AND expected<>rev THEN RAISE EXCEPTION 'classification revision changed' USING ERRCODE='40001'; END IF;
 IF org NOT IN ('manual','agent','rule','import','suggestion','dictionary') OR length(op) NOT BETWEEN 1 AND 100 OR length(req)<>64 THEN RAISE EXCEPTION 'classification command invalid'; END IF;
 PERFORM set_config('themis.classification_origin',org,true),set_config('themis.classification_actor',CASE WHEN org IN('manual','agent','dictionary') THEN own::text ELSE '' END,true),
 set_config('themis.classification_operation',op,true),set_config('themis.classification_request',req,true),set_config('themis.classification_rule','',true),set_config('themis.classification_suggestion','',true);
 RETURN false;
END $$;
CREATE FUNCTION classification_finish(own BIGINT,tid BIGINT) RETURNS VOID LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO classification_operations(owner_id,transaction_id,operation_id,request_hash,revision)
 SELECT own,tid,current_setting('themis.classification_operation'),current_setting('themis.classification_request'),
 COALESCE((SELECT revision FROM classification_state WHERE owner_id=own AND transaction_id=tid),0);
END $$;

CREATE FUNCTION classification_set_locks(own BIGINT,tid BIGINT,dl BOOLEAN,tl BOOLEAN) RETURNS VOID LANGUAGE plpgsql AS $$
DECLARE before JSONB;
BEGIN
 PERFORM id FROM transactions WHERE owner_id=own AND id=tid FOR UPDATE;
 before:=classification_snapshot(own,tid);
 IF before IS NULL THEN RAISE EXCEPTION 'classification transaction unavailable' USING ERRCODE='P0002'; END IF;
 IF COALESCE(current_setting('themis.classification_origin',true),'') NOT IN ('manual','agent') THEN RAISE EXCEPTION 'explicit decision required'; END IF;
 IF (dl IS NULL OR dl=(before->>'domain_locked')::boolean) AND (tl IS NULL OR tl=(before->>'tags_locked')::boolean) THEN RETURN; END IF;
 INSERT INTO classification_state(owner_id,transaction_id,domain_locked,tags_locked) VALUES(own,tid,COALESCE(dl,false),COALESCE(tl,false))
 ON CONFLICT(owner_id,transaction_id) DO UPDATE SET domain_locked=COALESCE(dl,classification_state.domain_locked),tags_locked=COALESCE(tl,classification_state.tags_locked);
 PERFORM classification_record(own,tid,'locks_changed',before,'explicit_lock_decision');
END $$;

CREATE FUNCTION classification_tag_decide(own BIGINT,tid BIGINT,tag BIGINT,wanted TEXT) RETURNS VOID LANGUAGE plpgsql AS $$
DECLARE old_dec TEXT; before JSONB; rev BIGINT;
BEGIN
 PERFORM id FROM transactions WHERE id=tid AND owner_id=own FOR UPDATE;
 IF NOT FOUND OR NOT EXISTS(SELECT 1 FROM tags WHERE id=tag AND owner_id=own) THEN RAISE EXCEPTION 'classification tag unavailable'; END IF;
 SELECT decision INTO old_dec FROM classification_tag_decisions WHERE owner_id=own AND transaction_id=tid AND tag_id=tag;
 IF old_dec=wanted THEN RETURN; END IF;
 IF COALESCE(current_setting('themis.classification_origin',true),'') NOT IN ('manual','agent') THEN RAISE EXCEPTION 'explicit tag decision required' USING ERRCODE='40001'; END IF;
 IF wanted NOT IN ('added','removed') THEN RAISE EXCEPTION 'classification tag decision invalid'; END IF;
 before:=classification_snapshot(own,tid);
 SELECT COALESCE((SELECT revision FROM classification_state WHERE owner_id=own AND transaction_id=tid),0)+1 INTO rev;
 INSERT INTO classification_tag_decisions(owner_id,transaction_id,tag_id,decision,revision,origin)
 VALUES(own,tid,tag,wanted,rev,COALESCE(NULLIF(current_setting('themis.classification_origin',true),''),'legacy_sql'))
 ON CONFLICT(owner_id,transaction_id,tag_id) DO UPDATE SET decision=EXCLUDED.decision,revision=EXCLUDED.revision,origin=EXCLUDED.origin,decided_at=now();
 PERFORM classification_record(own,tid,'tag_decision',before,'explicit_tag_'||wanted,tag);
END $$;

CREATE FUNCTION classification_transaction_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE before JSONB; locked BOOLEAN; explicit BOOLEAN;
BEGIN
 IF NEW.owner_id IS NULL THEN RETURN NEW; END IF;
 IF NEW.domain_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM domains WHERE id=NEW.domain_id AND owner_id=NEW.owner_id) THEN RAISE EXCEPTION 'classification foreign domain'; END IF;
 IF TG_OP='INSERT' THEN
  IF NEW.domain_id IS NOT NULL OR NEW.classification_source<>'unknown' THEN
   before:=classification_snapshot(NEW.owner_id,NEW.id)||jsonb_build_object('domain_id',NULL,'source','unknown');
   PERFORM classification_record(NEW.owner_id,NEW.id,'initial_classification',before,'new_transaction_assignment');
  END IF;
  RETURN NEW;
 END IF;
 IF NEW.domain_id IS NOT DISTINCT FROM OLD.domain_id AND NEW.classification_source IS NOT DISTINCT FROM OLD.classification_source THEN RETURN NEW; END IF;
 explicit:=COALESCE(current_setting('themis.classification_origin',true),'') IN ('manual','agent');
 SELECT domain_locked INTO locked FROM classification_state WHERE owner_id=OLD.owner_id AND transaction_id=OLD.id;
 IF NOT explicit AND (COALESCE(locked,false) OR (OLD.classification_source='manual' AND NEW.classification_source<>'manual')) THEN
  RAISE EXCEPTION 'classification domain protected' USING ERRCODE='40001';
 END IF;
 before:=classification_snapshot(OLD.owner_id,OLD.id)||jsonb_build_object('domain_id',OLD.domain_id,'source',OLD.classification_source);
 PERFORM classification_record(NEW.owner_id,NEW.id,'domain_changed',before,'domain_or_source_changed');
 RETURN NEW;
END $$;
-- AFTER triggers can observe the final bank row; raising rolls back the write.
CREATE TRIGGER classification_transaction_insert AFTER INSERT ON transactions FOR EACH ROW EXECUTE FUNCTION classification_transaction_guard();
CREATE TRIGGER classification_transaction_update AFTER UPDATE OF domain_id,classification_source ON transactions FOR EACH ROW EXECUTE FUNCTION classification_transaction_guard();

CREATE FUNCTION classification_tag_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE own BIGINT; tid BIGINT; tag BIGINT; explicit BOOLEAN; before JSONB;
BEGIN
 tid:=CASE WHEN TG_OP='DELETE' THEN OLD.transaction_id ELSE NEW.transaction_id END;
 tag:=CASE WHEN TG_OP='DELETE' THEN OLD.tag_id ELSE NEW.tag_id END;
 SELECT owner_id INTO own FROM transactions WHERE id=tid FOR UPDATE;
 IF own IS NULL THEN IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF; END IF;
 IF NOT EXISTS(SELECT 1 FROM tags WHERE id=tag AND owner_id=own) THEN RAISE EXCEPTION 'classification foreign tag'; END IF;
 explicit:=COALESCE(current_setting('themis.classification_origin',true),'') IN ('manual','agent');
 IF TG_WHEN='BEFORE' THEN
  IF NOT explicit AND (EXISTS(SELECT 1 FROM classification_state WHERE owner_id=own AND transaction_id=tid AND tags_locked)
   OR (TG_OP='INSERT' AND EXISTS(SELECT 1 FROM classification_tag_decisions WHERE owner_id=own AND transaction_id=tid AND tag_id=tag AND decision='removed'))) THEN RETURN NULL; END IF;
  IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
 END IF;
 before:=classification_snapshot(own,tid);
 -- Reconstruct just the immediately previous ID set, never bank text/amounts.
 IF TG_OP='INSERT' THEN before:=jsonb_set(before,'{tag_ids}',COALESCE((SELECT jsonb_agg(value ORDER BY value::bigint) FROM jsonb_array_elements(before->'tag_ids') WHERE value::bigint<>tag),'[]'));
 ELSE before:=jsonb_set(before,'{tag_ids}',(SELECT jsonb_agg(x ORDER BY x::bigint) FROM (SELECT value x FROM jsonb_array_elements(before->'tag_ids') UNION SELECT to_jsonb(tag)) ids)); END IF;
 PERFORM classification_record(own,tid,CASE WHEN TG_OP='INSERT' THEN 'tag_added' ELSE 'tag_removed' END,before,'tag_link_changed',tag);
 IF explicit THEN PERFORM classification_tag_decide(own,tid,tag,CASE WHEN TG_OP='INSERT' THEN 'added' ELSE 'removed' END); END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
END $$;
CREATE TRIGGER classification_tag_before BEFORE INSERT OR DELETE ON transaction_tags FOR EACH ROW EXECUTE FUNCTION classification_tag_guard();
CREATE TRIGGER classification_tag_after AFTER INSERT OR DELETE ON transaction_tags FOR EACH ROW EXECUTE FUNCTION classification_tag_guard();

CREATE FUNCTION classification_dictionary_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE rowid BIGINT; before JSONB;
BEGIN
 IF TG_OP='DELETE' THEN
  IF NOT EXISTS(SELECT 1 FROM users WHERE id=OLD.owner_id) THEN RETURN OLD; END IF;
  IF EXISTS(SELECT 1 FROM transaction_tags WHERE tag_id=OLD.id) OR EXISTS(SELECT 1 FROM classification_tag_decisions WHERE owner_id=OLD.owner_id AND tag_id=OLD.id) THEN
   RAISE EXCEPTION 'classification tag referenced; deletion refused' USING ERRCODE='40001'; END IF;
  RETURN OLD;
 END IF;
 IF NEW.name IS NOT DISTINCT FROM OLD.name THEN RETURN NEW; END IF;
 FOR rowid IN SELECT id FROM transactions WHERE owner_id=OLD.owner_id AND (id IN(SELECT transaction_id FROM transaction_tags WHERE tag_id=OLD.id) OR id IN(SELECT transaction_id FROM classification_tag_decisions WHERE tag_id=OLD.id AND owner_id=OLD.owner_id)) ORDER BY id FOR UPDATE LOOP
  before:=classification_snapshot(OLD.owner_id,rowid);
  PERFORM classification_record(OLD.owner_id,rowid,'tag_renamed',before,'stable_tag_identity_preserved',OLD.id);
 END LOOP;
 RETURN NEW;
END $$;
CREATE TRIGGER classification_dictionary BEFORE UPDATE OF name OR DELETE ON tags FOR EACH ROW EXECUTE FUNCTION classification_dictionary_guard();

CREATE FUNCTION classification_rule_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.owner_id IS NULL THEN RETURN NEW; END IF;
 IF TG_OP='UPDATE' AND
 (CASE WHEN left(NEW.pattern,3)='re:' THEN NEW.pattern ELSE upper(NEW.pattern) END) IS NOT DISTINCT FROM
 (CASE WHEN left(OLD.pattern,3)='re:' THEN OLD.pattern ELSE upper(OLD.pattern) END)
 AND NEW.domain_id IS NOT DISTINCT FROM OLD.domain_id
 AND (SELECT array_agg(v ORDER BY v) FROM (SELECT DISTINCT lower(btrim(x)) v FROM unnest(string_to_array(NEW.tags,',')) x WHERE btrim(x)<>'') q)
 IS NOT DISTINCT FROM (SELECT array_agg(v ORDER BY v) FROM (SELECT DISTINCT lower(btrim(x)) v FROM unnest(string_to_array(OLD.tags,',')) x WHERE btrim(x)<>'') q)
 THEN RETURN NEW; END IF;
 INSERT INTO classification_rule_versions(owner_id,rule_id,revision) VALUES(NEW.owner_id,NEW.id,1)
 ON CONFLICT(owner_id,rule_id) DO UPDATE SET revision=classification_rule_versions.revision+1;
 RETURN NEW;
END $$;
CREATE TRIGGER classification_rules AFTER INSERT OR UPDATE OF pattern,domain_id,tags ON rules FOR EACH ROW EXECUTE FUNCTION classification_rule_version();

CREATE FUNCTION classification_event_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 -- Account erasure may remove its metadata; ordinary journal edits cannot.
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM users WHERE id=OLD.owner_id) THEN RETURN OLD; END IF;
 RAISE EXCEPTION 'classification history immutable';
END $$;
CREATE TRIGGER classification_events_immutable BEFORE UPDATE OR DELETE ON classification_events FOR EACH ROW EXECUTE FUNCTION classification_event_immutable();

CREATE FUNCTION classification_link_update_refused() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN IF NEW.transaction_id<>OLD.transaction_id OR NEW.tag_id<>OLD.tag_id THEN
 RAISE EXCEPTION 'classification link update requires explicit removal and addition'; END IF; RETURN NEW; END $$;
CREATE TRIGGER classification_link_update BEFORE UPDATE ON transaction_tags FOR EACH ROW EXECUTE FUNCTION classification_link_update_refused();

CREATE FUNCTION classification_domain_delete_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM users WHERE id=OLD.owner_id) THEN RETURN OLD; END IF;
 IF EXISTS(SELECT 1 FROM transactions WHERE owner_id=OLD.owner_id AND domain_id IN(SELECT id FROM domains WHERE owner_id=OLD.owner_id AND (id=OLD.id OR parent_id=OLD.id))) THEN
 RAISE EXCEPTION 'classification domain in use; deletion refused' USING ERRCODE='40001'; END IF;
 RETURN OLD;
END $$;
CREATE TRIGGER classification_domain_delete BEFORE DELETE ON domains FOR EACH ROW EXECUTE FUNCTION classification_domain_delete_guard();
