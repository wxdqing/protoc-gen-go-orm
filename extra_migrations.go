package main

func extraMigrationsForMessage(msg MessageDesc) []string {
	if !msg.OrmOptions.IsTable {
		return nil
	}
	table := msg.TableName
	if table == "" {
		table = toSnakeCase(msg.Name)
	}
	var out []string
	out = append(out, partialIndexMigrations(table, collectPartialIndexes(msg))...)
	out = append(out, foreignKeyMigrations(table, collectForeignKeys(msg))...)
	return out
}
