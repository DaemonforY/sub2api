-- 请求内容日志: the 遮盖密钥 switch (settings key content_log_redact_secrets) is audited as action
-- 'config' (target_user_id 0) in the same access trail.
ALTER TABLE content_log_access_logs DROP CONSTRAINT IF EXISTS content_log_access_logs_action_check;
ALTER TABLE content_log_access_logs ADD CONSTRAINT content_log_access_logs_action_check
    CHECK (action IN ('enable', 'disable', 'view', 'config'));
