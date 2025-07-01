# File Naming Conventions

## Standardized Naming Rules

### General Rules
1. **Use hyphens (-) instead of underscores (_)** for multi-word file names
2. **Use lowercase letters** for all file names
3. **Be descriptive** - avoid generic names like `basic.md`, `advance.md`
4. **Include context** when files might have similar names across directories
5. **Version numbers** should be descriptive: `v1` → `patterns-v1`

### Specific Patterns

#### Topic-Specific Naming
- **Query files**: Add suffix like `-queries`, `-patterns`
  - `search.md` → `search-queries.md` (basic queries)
  - `search.md` → `full-text-search.md` (specialized)
- **Schema files**: Add suffix like `-schema`, `-patterns`
  - `soft-delete.md` → `soft-delete-schema.md`
- **Examples**: Add suffix like `-examples`, `-implementation`
  - `search.md` → `search-examples.md`

#### Technical Naming
- **Database-specific**: Include database name
  - `mysql_8_uuid_v4.md` → `mysql-uuid-v4.md`
  - `mysql_gipk.md` → `mysql-gipk.md`
- **Version files**: Be descriptive about content
  - `temporal.v1.md` → `temporal-patterns-v1.md`
  - `wallet.v2.md` → `wallet-schema-v2.md`

#### Incident/Troubleshooting Files
- **Include descriptive names**: 
  - `001_mysql.md` → `mysql-incident-001.md`
  - `002_lock_wait_timeout.md` → `lock-wait-timeout.md`

### Examples of Renamed Files

| Old Name | New Name | Reason |
|----------|----------|---------|
| `access_control.md` | `access-control.md` | Hyphen consistency |
| `chasm_and_fan_traps.md` | `chasm-and-fan-traps.md` | Hyphen consistency |
| `001-bulk-update.md` | `bulk-update-patterns.md` | Descriptive naming |
| `advance.md` | `advanced-patterns.md` | Clear, descriptive |
| `search.md` (basic) | `search-queries.md` | Context-specific |
| `search.md` (specialized) | `full-text-search.md` | Context-specific |
| `temporal.v1.md` | `temporal-patterns-v1.md` | Descriptive versioning |

### Directory-Specific Rules

#### `/query-patterns/`
- Basic queries: `-queries` suffix
- Advanced patterns: `-patterns` suffix
- Performance: `-optimization` suffix

#### `/schema-design/`
- Schema patterns: `-schema` or `-patterns` suffix
- Relationships: `-relationships` suffix
- Constraints: `-constraints` suffix

#### `/examples/`
- Schema examples: `-schema` suffix
- Implementation guides: `-implementation` suffix
- Case studies: descriptive names

#### `/operations/`
- Tools: `-tools` suffix
- Incidents: descriptive names with context
- Procedures: `-procedures` or `-guide` suffix

### Validation

Run this command to check for naming consistency:
```bash
# Find files that might need renaming
find . -name "*.md" -type f | grep -E "(_.+\.md$|^.*[0-9]{3}.*\.md$|basic\.md$|advance\.md$)"
```

### Benefits
1. **Consistency**: All files follow the same naming pattern
2. **Discoverability**: Descriptive names make content easier to find
3. **Maintainability**: Clear naming makes organization easier
4. **No Conflicts**: Context-specific naming prevents duplicate file names
5. **Professional**: Consistent, clean naming improves documentation quality
