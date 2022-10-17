package google_sql

import (
	"errors"
	"fmt"
	"strconv"
)

type flagValidator = func(value string) error

const maxFloat32 float64 = 3.4028235e+38
const maxInt32 = 2147483647
const eightk = 8192

func ValidateFlag(key string, value string) error {
	validatorFunc := validators[key]
	if validatorFunc == nil {
		return fmt.Errorf("couldn't find validator for instance flag '%s'", key)
	}
	return validatorFunc(value)
}

var validators = map[string]flagValidator{
	"auto_explain.log_analyze":                      toBool,
	"auto_explain.log_buffers":                      toBool,
	"auto_explain.log_min_duration":                 intWithinRange(-1, maxInt32),
	"auto_explain.log_format":                       inEnum([]string{"text", "xml", "json", "yaml"}),
	"auto_explain.log_level":                        inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "debug", "info", "notice", "warning", "log"}),
	"auto_explain.log_nested_statements":            toBool,
	"auto_explain.log_settings":                     toBool,
	"auto_explain.log_timing":                       toBool,
	"auto_explain.log_triggers":                     toBool,
	"auto_explain.log_wal":                          toBool,
	"auto_explain.log_verbose":                      toBool,
	"auto_explain.sample_rate":                      floatWithinRange(0.0, 1.0),
	"autovacuum":                                    toBool,
	"autovacuum_analyze_scale_factor":               floatWithinRange(0.0, 100.0),
	"autovacuum_analyze_threshold":                  intWithinRange(0, maxInt32),
	"autovacuum_freeze_max_age":                     intWithinRange(0, 2000000000),
	"autovacuum_max_workers":                        intWithinRange(1, 262142),
	"autovacuum_multixact_freeze_max_age":           intWithinRange(10000, 2000000000),
	"autovacuum_naptime":                            intWithinRange(1, 2147483),
	"autovacuum_vacuum_cost_delay":                  intWithinRange(-1, 100),
	"autovacuum_vacuum_cost_limit":                  intWithinRange(-1, 10000),
	"autovacuum_vacuum_scale_factor":                floatWithinRange(0.0, 100.0),
	"autovacuum_vacuum_threshold":                   intWithinRange(0, maxInt32),
	"autovacuum_work_mem":                           intWithinRange(-1, maxInt32),
	"checkpoint_completion_target":                  floatWithinRange(0.0, 1.0),
	"checkpoint_timeout":                            intWithinRange(30, 86400),
	"checkpoint_warning":                            intWithinRange(0, maxInt32),
	"cloudsql.allow_passwordless_local_connections": toBool,
	"cloudsql.enable_auto_explain":                  toBool,
	"cloudsql.enable_pgaudit":                       toBool,
	"cloudsql.enable_pg_bigm":                       toBool,
	"cloudsql.enable_pg_cron":                       toBool,
	"cloudsql.enable_pg_hint_plan":                  toBool,
	"cloudsql.enable_pglogical":                     toBool,
	"cloudsql.iam_authentication":                   toBool,
	"cloudsql.logical_decoding":                     toBool,
	"cloudsql.pg_shadow_select_role":                notEmpty,
	"commit_delay":                                  intWithinRange(0, 100000),
	"commit_siblings":                               intWithinRange(0, 1000),
	"constraint_exclusion":                          inEnum([]string{"partition", "on", "off"}),
	"cpu_index_tuple_cost":                          floatWithinRange(0.0, maxFloat32),
	"cpu_operator_cost":                             floatWithinRange(0.0, maxFloat32),
	"cpu_tuple_cost":                                floatWithinRange(0.0, maxFloat32),
	"cron.database_name":                            notEmpty,
	"cron.log_statement":                            toBool,
	"cron.log_run":                                  toBool,
	"cron.max_running_jobs":                         intWithinRange(0, maxInt32),
	"cron.log_min_messages":                         inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "debug", "info", "notice", "warning", "error", "log", "fatal", "panic"}),
	"cursor_tuple_fraction":                         floatWithinRange(0.0, 1.0),
	"deadlock_timeout":                              intWithinRange(1, maxInt32),
	"effective_cache_size":                          unit(eightk),
	"default_statistics_target":                     intWithinRange(1, 10000),
	"default_tablespace":                            notEmpty,
	"default_transaction_deferrablze":               toBool,
	"default_transaction_isolation":                 inEnum([]string{"serializable", "repeatable read", "read committed", "read uncommitted"}),
	"enable_bitmapscan":                             toBool,
	"enable_hashagg":                                toBool,
	"enable_hashjoin":                               toBool,
	"enable_indexonlyscan":                          toBool,
	"enable_indexscan":                              toBool,
	"enable_material":                               toBool,
	"enable_mergejoin":                              toBool,
	"enable_nestloop":                               toBool,
	"enable_seqscan":                                toBool,
	"enable_sort":                                   toBool,
	"enable_tidscan":                                toBool,
	"force_parallel_mode":                           inEnum([]string{"off", "on", "regress"}),
	"from_collapse_limit":                           intWithinRange(1, maxInt32),
	"geqo":                                          toBool,
	"geqo_effort":                                   intWithinRange(1, 10),
	"geqo_generations":                              intWithinRange(0, maxInt32),
	"geqo_pool_size":                                intWithinRange(0, maxInt32),
	"geqo_seed":                                     floatWithinRange(0.0, 1.0),
	"geqo_selection_bias":                           floatWithinRange(1.5, 2.0),
	"geqo_threshold":                                intWithinRange(2, maxInt32),
	"gin_fuzzy_search_limit":                        intWithinRange(0, maxInt32),
	"gin_pending_list_limit":                        intWithinRange(64, maxInt32),
	"hot_standby_feedback":                          toBool,
	"huge_pages":                                    inEnum([]string{"try", "off"}),
	"idle_in_transaction_session_timeout":           intWithinRange(0, maxInt32),
	"join_collapse_limit":                           intWithinRange(1, maxInt32),
	"lock_timeout":                                  intWithinRange(0, maxInt32),
	"log_autovacuum_min_duration":                   intWithinRange(-1, maxInt32),
	"log_checkpoints":                               toBool,
	"log_connections":                               toBool,
	"log_disconnections":                            toBool,
	"log_duration":                                  toBool,
	"log_error_verbosity":                           inEnum([]string{"terse", "default", "verbose"}),
	"log_executor_stats":                            toBool,
	"log_hostname":                                  toBool,
	"log_lock_waits":                                toBool,
	"log_min_duration_statement":                    intWithinRange(-1, maxInt32),
	"log_min_error_statement":                       inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "info", "notice", "warning", "error", "log", "fatal", "panic"}),
	"log_min_messages":                              inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "info", "notice", "warning", "error", "log", "fatal", "panic"}),
	"log_parser_stats":                              toBool,
	"log_planner_stats":                             toBool,
	"log_replication_commands":                      toBool,
	"log_statement":                                 inEnum([]string{"none", "ddl", "mod", "all"}),
	"log_statement_stats":                           toBool,
	"log_temp_files":                                intWithinRange(-1, maxInt32),
	"maintenance_work_mem":                          intWithinRange(1024, maxInt32),
	"max_connections":                               intWithinRange(1, maxInt32),
	"max_locks_per_transaction":                     intWithinRange(10, maxInt32),
	"max_logical_replication_workers":               intWithinRange(4, 8192),
	"max_parallel_maintenance_workers":              intWithinRange(0, maxInt32),
	"max_parallel_workers":                          intWithinRange(0, maxInt32),
	"max_parallel_workers_per_gather":               intWithinRange(0, maxInt32),
	"max_pred_locks_per_page":                       intWithinRange(0, maxInt32),
	"max_pred_locks_per_relation":                   intWithinRange(-2147483648, maxInt32),
	"max_pred_locks_per_transaction":                intWithinRange(0, 1048576),
	"max_prepared_transactions":                     intWithinRange(0, maxInt32),
	"max_replication_slots":                         intWithinRange(10, maxInt32),
	"max_standby_archive_delay":                     intWithinRange(-1, maxInt32),
	"max_standby_streaming_delay":                   intWithinRange(-1, maxInt32),
	"max_sync_workers_per_subscription":             intWithinRange(2, 64),
	"max_wal_senders":                               intWithinRange(10, maxInt32),
	"max_wal_size":                                  intWithinRange(2, maxInt32),
	"max_worker_processes":                          intWithinRange(8, maxInt32),
	"min_parallel_relation_size":                    unit(eightk),
	"min_wal_size":                                  unit(16777216),
	"old_snapshot_threshold":                        intWithinRange(0, 86400),
	"parallel_setup_cost":                           floatWithinRange(0.0, maxFloat32),
	"parallel_tuple_cost":                           floatWithinRange(0.0, maxFloat32),
	"password_encryption":                           inEnum([]string{"md5", "scram-sha-256"}),
	"pg_bigm.enable_recheck":                        toBool,
	"pg_bigm.gin_key_limit":                         intWithinRange(0, maxInt32),
	"pg_bigm.pg_bigm.similarity_limit":              floatWithinRange(0.0, 1.0),
	"pg_hint_plan.enable_hint":                      toBool,
	"pg_hint_plan.debug_print":                      inEnum([]string{"off", "on", "detailed", "verbose", "0", "1", "2", "3", "no", "yes", "false", "true"}),
	"pg_hint_plan.parse_messages":                   inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "debug", "info", "notice", "warning", "error", "log"}),
	"pg_hint_plan.message_level":                    inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "debug", "info", "notice", "warning", "error", "log"}),
	"pg_hint_plan.enable_hint_table":                toBool,
	"pglogical.batch_inserts":                       toBool,
	"pglogical.conflict_log_level":                  notEmpty,
	"pglogical.conflict_resolution":                 inEnum([]string{"error", "apply_remote", "keep_local", "last_update_wins", "first_update_wins"}),
	"pglogical.extra_connection_options":            notEmpty,
	"pglogical.synchronous_commit":                  toBool,
	"pglogical.pglogical.use_spi":                   toBool,
	"pg_stat_statements.max":                        intWithinRange(100, maxInt32),
	"pg_stat_statements.save":                       toBool,
	"pg_stat_statements.track":                      inEnum([]string{"none", "top", "all"}),
	"pg_stat_statements.track_utility":              toBool,
	"pgaudit.log":                                   inEnum([]string{"read", "write", "function", "role", "ddl", "misc", "misc_set", "all", "none"}),
	"pgaudit.log_catalog":                           toBool,
	"pgaudit.log_level":                             inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "info", "notice", "warning", "error", "log"}),
	"pgaudit.log_parameter":                         toBool,
	"pgaudit.pgaudit.log_relation":                  toBool,
	"pgaudit.log_statement_once":                    toBool,
	"pgaudit.role":                                  notEmpty,
	"random_page_cost":                              floatWithinRange(0.0, maxFloat32),
	"session_replication_role":                      inEnum([]string{"origin", "replica", "local"}),
	"replacement_sort_tuples":                       intWithinRange(0, maxInt32),
	"shared_buffers":                                unit(eightk),
	"ssl_max_protocol_version":                      inEnum([]string{"TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"}),
	"ssl_min_protocol_version":                      inEnum([]string{"TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"}),
	"standard_conforming_strings":                   toBool,
	"synchronize_seqscans":                          toBool,
	"tcp_keepalives_count":                          intWithinRange(0, maxInt32),
	"tcp_keepalives_idle":                           intWithinRange(0, maxInt32),
	"tcp_keepalives_interval":                       intWithinRange(0, maxInt32),
	"temp_buffers":                                  unit(eightk),
	"temp_file_limit":                               intWithinRange(1048576, maxInt32),
	"trace_notify":                                  toBool,
	"trace_recovery_messages":                       inEnum([]string{"debug5", "debug4", "debug3", "debug2", "debug1", "log", "notice", "warning", "error"}),
	"trace_sort":                                    toBool,
	"track_activities":                              toBool,
	"track_activity_query_size":                     intWithinRange(100, 102400),
	"track_commit_timestamp":                        toBool,
	"track_counts":                                  toBool,
	"track_functions":                               inEnum([]string{"none", "pl", "all"}),
	"track_io_timing":                               toBool,
	"vacuum_cost_delay":                             intWithinRange(0, 100),
	"vacuum_cost_limit":                             intWithinRange(1, 10000),
	"vacuum_freeze_min_age":                         intWithinRange(0, 1000000000),
	"vacuum_freeze_table_age":                       intWithinRange(0, 2000000000),
	"vacuum_multixact_freeze_min_age":               intWithinRange(0, 1000000000),
	"vacuum_multixact_freeze_table_age":             intWithinRange(0, 2000000000),
	"wal_buffers":                                   unit(eightk),
	"wal_compression":                               toBool,
	"wal_receiver_timeout":                          intWithinRange(0, maxInt32),
	"wal_sender_timeout":                            intWithinRange(0, maxInt32),
	"work_mem":                                      intWithinRange(64, maxInt32),
}

func intWithinRange(min int, max int) func(n string) error {
	return func(n string) error {
		i, err := toInt(n)
		if err != nil {
			return err
		}
		if i < min || i > max {
			return fmt.Errorf("%d is not between %d and %d", i, min, max)
		}
		return nil
	}
}

func floatWithinRange(min float64, max float64) func(n string) error {
	return func(n string) error {
		f, err := toFloat(n)
		if err != nil {
			return err
		}
		if f < min || f > max {
			return fmt.Errorf("%f is not between %f and %f", f, min, max)
		}
		return nil
	}
}

func inEnum(allowedVals []string) func(val string) error {
	return func(val string) error {
		for _, v := range allowedVals {
			if val == v {
				return nil
			}
		}
		return fmt.Errorf("%s is not in %v", val, allowedVals)
	}
}

func unit(unitSize int) func(n string) error {
	return func(n string) error {
		i, err := toInt(n)
		if err != nil {
			return err
		}
		if i%unitSize != 0 {
			return fmt.Errorf("%d is not a unit of %d", i, unitSize)
		}
		return nil
	}
}

func notEmpty(str string) error {
	if str == "" {
		return errors.New("value cannot be empty")
	}
	return nil
}

func toBool(str string) error {
	_, err := strconv.ParseBool(str)
	if err != nil {
		return err
	}
	return nil
}

func toInt(str string) (int, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("expected an int, got '%s'", str)
	}
	return i, nil
}

func toFloat(str string) (float64, error) {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("expected a float, got '%s'", str)
	}
	return f, nil
}
