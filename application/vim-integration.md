# Vim Database Integration

Integration patterns and workflows for using Vim with database development, including PostgreSQL, MySQL, and other database systems.

## 📚 Table of Contents

- [Overview](#overview)
- [PostgreSQL Integration](#postgresql-integration)
- [MySQL Integration](#mysql-integration)
- [Database Clients in Vim](#database-clients-in-vim)
- [Query Development Workflow](#query-development-workflow)
- [Vim Plugins for Databases](#vim-plugins-for-databases)
- [Configuration Examples](#configuration-examples)
- [Best Practices](#best-practices)

## Overview

Vim can be effectively integrated with database development workflows, allowing developers to write, edit, and execute SQL queries directly from their editor. This integration is particularly useful for:

- **Rapid Query Development**: Write and test queries without leaving the editor
- **Schema Exploration**: Browse database structure and metadata
- **Query Formatting**: Automatic SQL formatting and syntax highlighting
- **Result Visualization**: View query results in formatted tables
- **Script Management**: Organize and version control database scripts

## PostgreSQL Integration

### Basic psql Integration
```bash
# Edit SQL in Vim and execute in psql
$ psql -d mydb
mydb=# \e

# This opens your default editor (set to vim with EDITOR=vim)
# Write your query, save and exit, psql will execute it
```

### Vim Configuration for PostgreSQL
```vim
" ~/.vimrc - PostgreSQL specific settings
augroup postgresql
  autocmd!
  autocmd BufRead,BufNewFile *.sql set filetype=sql
  autocmd FileType sql setlocal commentstring=--\ %s
  autocmd FileType sql setlocal expandtab tabstop=2 shiftwidth=2
augroup END

" PostgreSQL specific commands
command! -range PsqlExecute <line1>,<line2>w !psql -d mydb
nnoremap <leader>pe :PsqlExecute<CR>
vnoremap <leader>pe :PsqlExecute<CR>

" Function to execute current query block
function! ExecutePostgreSQLQuery()
  let save_cursor = getcurpos()
  normal! vip
  :'<,'>w !psql -d mydb
  call setpos('.', save_cursor)
endfunction

nnoremap <leader>pq :call ExecutePostgreSQLQuery()<CR>
```

### Advanced psql Integration
```bash
# Set up psql environment
export PSQL_EDITOR='vim'
export PSQL_PAGER='less -S'

# Create .psqlrc for better formatting
# ~/.psqlrc
\set QUIET 1
\pset null '[NULL]'
\pset linestyle unicode
\pset border 2
\x auto
\set COMP_KEYWORD_CASE upper
\set HISTSIZE 2000
\set PROMPT1 '%[%033[1m%]%M %n@%/%R%[%033[0m%]%# '
\set PROMPT2 '[more] %R > '
\unset QUIET

# Useful aliases
\set version 'SELECT version();'
\set extensions 'SELECT * FROM pg_available_extensions;'
\set conninfo 'SELECT usename, application_name, client_addr, state FROM pg_stat_activity;'
```

### PostgreSQL Query Templates
```sql
-- Template for complex queries (save as template.sql)
-- PostgreSQL Query Template
-- Database: 
-- Purpose: 
-- Author: 
-- Date: 

\timing on
\echo 'Starting query execution...'

WITH query_params AS (
  SELECT 
    '2023-01-01'::date as start_date,
    '2023-12-31'::date as end_date
),
main_query AS (
  SELECT 
    -- Add your main query here
    column1,
    column2
  FROM your_table t
  CROSS JOIN query_params p
  WHERE t.created_at BETWEEN p.start_date AND p.end_date
)
SELECT * FROM main_query
ORDER BY column1
LIMIT 100;

\echo 'Query completed.'
```

## MySQL Integration

### MySQL Client Integration
```vim
" MySQL specific configuration
augroup mysql
  autocmd!
  autocmd BufRead,BufNewFile *.mysql set filetype=mysql
  autocmd FileType mysql setlocal commentstring=--\ %s
augroup END

" MySQL execution commands
command! -range MysqlExecute <line1>,<line2>w !mysql -u username -p database_name
nnoremap <leader>me :MysqlExecute<CR>
vnoremap <leader>me :MysqlExecute<CR>

" Function to execute MySQL queries with better formatting
function! ExecuteMySQLQuery()
  let save_cursor = getcurpos()
  normal! vip
  :'<,'>w !mysql -u username -p -t database_name
  call setpos('.', save_cursor)
endfunction

nnoremap <leader>mq :call ExecuteMySQLQuery()<CR>
```

### MySQL Configuration
```bash
# ~/.my.cnf - MySQL client configuration
[mysql]
prompt="\\u@\\h [\\d]> "
pager="less -niSFX"
tee=/tmp/mysql.log
auto-rehash
safe-updates

[client]
default-character-set=utf8mb4
```

## Database Clients in Vim

### vim-dadbod Plugin
One of the most popular Vim database plugins:

```vim
" Install with vim-plug
Plug 'tpope/vim-dadbod'
Plug 'kristijanhusak/vim-dadbod-ui'
Plug 'kristijanhusak/vim-dadbod-completion'

" Configuration
let g:db_ui_save_location = '~/db_queries'
let g:db_ui_tmp_query_location = '~/db_queries/tmp'

" Database connections
let g:dbs = {
  \ 'dev_postgres': 'postgresql://user:pass@localhost:5432/dev_db',
  \ 'dev_mysql': 'mysql://user:pass@localhost:3306/dev_db',
  \ 'staging': 'postgresql://user:pass@staging-host:5432/staging_db'
  \ }

" Key mappings
nnoremap <leader>db :DBUI<CR>
nnoremap <leader>dc :DB g:dbs.dev_postgres 
vnoremap <leader>dc :DB g:dbs.dev_postgres<CR>
```

### Using vim-dadbod
```sql
-- Execute queries directly in Vim
-- Method 1: Use :DB command
:DB g:dbs.dev_postgres SELECT * FROM users LIMIT 10;

-- Method 2: Visual selection + DB command
SELECT 
  id,
  name,
  email,
  created_at
FROM users 
WHERE active = true
ORDER BY created_at DESC
LIMIT 10;
-- Select the query and press <leader>dc
```

### SQLComplete Plugin
```vim
" SQL completion and formatting
Plug 'vim-scripts/SQLComplete.vim'
Plug 'lifepillar/pgsql.vim'

" PostgreSQL syntax highlighting
let g:sql_type_default = 'pgsql'

" SQL formatting
Plug 'vim-scripts/SQLUtilities'
nnoremap <leader>sf :SQLUFormatter<CR>
vnoremap <leader>sf :SQLUFormatter<CR>
```

## Query Development Workflow

### Structured SQL Files
```sql
-- queries/user_analytics.sql
-- =============================================================================
-- User Analytics Queries
-- =============================================================================
-- Description: Collection of queries for user behavior analysis
-- Last Updated: 2023-12-01
-- Dependencies: users, sessions, events tables
-- =============================================================================

-- Query 1: Daily Active Users
-- Purpose: Get DAU for the last 30 days
\echo '=== Daily Active Users ==='
SELECT 
  DATE(created_at) as date,
  COUNT(DISTINCT user_id) as daily_active_users
FROM sessions 
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date;

-- Query 2: User Retention
-- Purpose: Calculate 7-day retention rate
\echo '=== User Retention Analysis ==='
WITH cohorts AS (
  SELECT 
    user_id,
    DATE(MIN(created_at)) as cohort_date
  FROM sessions
  GROUP BY user_id
),
user_activities AS (
  SELECT 
    s.user_id,
    c.cohort_date,
    DATE(s.created_at) as activity_date,
    DATE(s.created_at) - c.cohort_date as days_since_signup
  FROM sessions s
  JOIN cohorts c ON s.user_id = c.user_id
)
SELECT 
  cohort_date,
  COUNT(DISTINCT CASE WHEN days_since_signup = 0 THEN user_id END) as day_0_users,
  COUNT(DISTINCT CASE WHEN days_since_signup = 7 THEN user_id END) as day_7_users,
  ROUND(
    COUNT(DISTINCT CASE WHEN days_since_signup = 7 THEN user_id END) * 100.0 / 
    COUNT(DISTINCT CASE WHEN days_since_signup = 0 THEN user_id END), 2
  ) as retention_rate
FROM user_activities
GROUP BY cohort_date
ORDER BY cohort_date;
```

### Query Organization
```bash
# Directory structure for SQL files
db_queries/
├── schemas/
│   ├── create_tables.sql
│   ├── indexes.sql
│   └── constraints.sql
├── migrations/
│   ├── 001_initial_schema.sql
│   ├── 002_add_user_preferences.sql
│   └── 003_optimize_indexes.sql
├── analytics/
│   ├── user_metrics.sql
│   ├── revenue_analysis.sql
│   └── performance_monitoring.sql
├── maintenance/
│   ├── cleanup_old_data.sql
│   ├── reindex_tables.sql
│   └── vacuum_analyze.sql
└── templates/
    ├── query_template.sql
    └── migration_template.sql
```

### Vim SQL Snippets
```vim
" ~/.vim/after/ftplugin/sql.vim
" SQL snippets for faster development

" Common SELECT template
iabbrev sel SELECT<CR>  column1,<CR>  column2<CR>FROM table_name<CR>WHERE condition<CR>ORDER BY column1;<Esc>5k$

" CTE template  
iabbrev cte WITH query_name AS (<CR>  SELECT<CR>    columns<CR>  FROM table<CR>  WHERE condition<CR>)<CR>SELECT * FROM query_name;<Esc>6k$

" Window function template
iabbrev winf ROW_NUMBER() OVER (<CR>  PARTITION BY column<CR>  ORDER BY column<CR>) as row_num<Esc>2k$

" Index creation template
iabbrev idx CREATE INDEX CONCURRENTLY idx_table_column<CR>ON table_name (column_name);<Esc>k$

" Function to add SQL header
function! AddSQLHeader()
  let filename = expand('%:t:r')
  let date = strftime('%Y-%m-%d')
  call append(0, [
    \ '-- =============================================================================',
    \ '-- ' . toupper(filename),
    \ '-- =============================================================================',
    \ '-- Description: ',
    \ '-- Author: ' . $USER,
    \ '-- Date: ' . date,
    \ '-- Last Modified: ' . date,
    \ '-- =============================================================================',
    \ ''
  \ ])
  normal! 4G$
endfunction

command! SQLHeader call AddSQLHeader()
```

## Vim Plugins for Databases

### Essential Database Plugins
```vim
" Database interaction
Plug 'tpope/vim-dadbod'                    " Universal database interface
Plug 'kristijanhusak/vim-dadbod-ui'        " UI for vim-dadbod
Plug 'kristijanhusak/vim-dadbod-completion' " Auto-completion

" SQL syntax and formatting
Plug 'lifepillar/pgsql.vim'                " PostgreSQL syntax
Plug 'vim-scripts/SQLUtilities'            " SQL formatting utilities
Plug 'vim-scripts/Align'                   " Required by SQLUtilities

" Query result handling
Plug 'chrisbra/csv.vim'                    " CSV file handling for results
Plug 'mechatroner/rainbow_csv'             " Rainbow CSV highlighting

" Documentation and help
Plug 'vim-scripts/dbext.vim'               " Alternative database interface
```

### Plugin Configuration
```vim
" vim-dadbod configuration
let g:db_ui_auto_execute_table_helpers = 1
let g:db_ui_show_database_icon = 1
let g:db_ui_use_nerd_fonts = 1
let g:db_ui_force_echo_messages = 1

" PostgreSQL syntax highlighting
let g:sql_type_default = 'pgsql'
let g:pgsql_pl = ['python', 'javascript', 'sql']

" CSV plugin for query results
let g:csv_delim_test = ',;|'
let g:csv_no_conceal = 1

" Auto-completion for SQL
autocmd FileType sql setlocal omnifunc=vim_dadbod_completion#omni
autocmd FileType sql,mysql,plsql lua require('cmp').setup.buffer({ sources = {{ name = 'vim-dadbod-completion' }} })
```

## Configuration Examples

### Complete .vimrc Database Setup
```vim
" Database development configuration
set nocompatible
filetype plugin indent on
syntax enable

" Plugin management (vim-plug)
call plug#begin('~/.vim/plugged')
  Plug 'tpope/vim-dadbod'
  Plug 'kristijanhusak/vim-dadbod-ui'
  Plug 'kristijanhusak/vim-dadbod-completion'
  Plug 'lifepillar/pgsql.vim'
  Plug 'vim-scripts/SQLUtilities'
  Plug 'chrisbra/csv.vim'
call plug#end()

" Database connections
let g:dbs = {
  \ 'local_pg': 'postgresql://postgres:password@localhost:5432/mydb',
  \ 'local_mysql': 'mysql://root:password@localhost:3306/mydb',
  \ 'dev': $DATABASE_DEV_URL,
  \ 'staging': $DATABASE_STAGING_URL
  \ }

" SQL file settings
augroup sql_settings
  autocmd!
  autocmd FileType sql setlocal expandtab tabstop=2 shiftwidth=2
  autocmd FileType sql setlocal commentstring=--\ %s
  autocmd FileType sql setlocal foldmethod=indent
  autocmd FileType sql nnoremap <buffer> <leader>r :DB<CR>
  autocmd FileType sql vnoremap <buffer> <leader>r :DB<CR>
augroup END

" Key mappings
nnoremap <leader>db :DBUI<CR>
nnoremap <leader>dt :DBUIToggle<CR>
nnoremap <leader>df :DBUIFindBuffer<CR>
nnoremap <leader>dr :DBUIRenameBuffer<CR>

" SQL formatting
vnoremap <leader>sf :SQLUFormatter<CR>
nnoremap <leader>sf :SQLUFormatter<CR>

" Query execution shortcuts
nnoremap <leader>qr :DB g:dbs.local_pg 
vnoremap <leader>qr :DB g:dbs.local_pg<CR>

" Function to switch database context
function! SwitchDB(db_key)
  let g:db = g:dbs[a:db_key]
  echo "Switched to database: " . a:db_key
endfunction

command! -nargs=1 SwitchDB call SwitchDB(<q-args>)
nnoremap <leader>sd :SwitchDB 
```

### Database-Specific Vim Scripts
```vim
" ~/.vim/after/ftplugin/sql.vim
" PostgreSQL specific settings

" Set PostgreSQL as default SQL dialect
let b:sql_type_override='pgsql' | set ft=sql

" PostgreSQL specific snippets
iabbrev jsonb_extract jsonb_extract_path_text(column, 'key')
iabbrev array_agg array_agg(column ORDER BY column)
iabbrev window_lag LAG(column, 1) OVER (PARTITION BY group_col ORDER BY order_col)
iabbrev upsert INSERT INTO table (columns) VALUES (values) ON CONFLICT (column) DO UPDATE SET

" Query optimization helpers
command! Explain let @q = 'EXPLAIN (ANALYZE, BUFFERS) ' . @q
command! ExplainPlan let @q = 'EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) ' . @q

" Performance monitoring queries
command! SlowQueries read ~/.vim/sql_templates/slow_queries.sql
command! IndexUsage read ~/.vim/sql_templates/index_usage.sql
command! TableSizes read ~/.vim/sql_templates/table_sizes.sql
```

## Best Practices

### 1. Environment Management
```bash
# Use environment variables for database connections
export DATABASE_DEV_URL="postgresql://user:pass@dev-host:5432/mydb"
export DATABASE_STAGING_URL="postgresql://user:pass@staging-host:5432/mydb"
export DATABASE_PROD_URL="postgresql://user:pass@prod-host:5432/mydb"

# Different configurations for different environments
export PSQL_DEV="psql $DATABASE_DEV_URL"
export PSQL_STAGING="psql $DATABASE_STAGING_URL"

# Aliases for quick access
alias psql-dev='psql $DATABASE_DEV_URL'
alias psql-staging='psql $DATABASE_STAGING_URL'
```

### 2. Query Safety
```vim
" Add safety checks for destructive operations
function! SafetyCheck()
  let line = getline('.')
  if line =~? '^\s*\(delete\|drop\|truncate\|update\)\>'
    let response = input('Destructive operation detected! Continue? (y/N): ')
    if response !=? 'y'
      echo "\nOperation cancelled."
      return 0
    endif
  endif
  return 1
endfunction

" Hook into query execution
nnoremap <buffer> <leader>qr :if SafetyCheck()<CR>:DB<CR>:endif<CR>
```

### 3. Query Organization
```sql
-- Use consistent query structure
-- 1. Header with metadata
-- 2. Parameter definitions
-- 3. CTEs for complex logic
-- 4. Main query
-- 5. Output formatting

-- Query: User Engagement Analysis
-- Purpose: Calculate user engagement metrics
-- Parameters: date_range, user_segment
-- Output: CSV format for reporting

\set start_date '2023-01-01'
\set end_date '2023-12-31'

WITH date_range AS (
  SELECT 
    :'start_date'::date as start_date,
    :'end_date'::date as end_date
),
-- Additional CTEs...
final_result AS (
  -- Main query logic
)
SELECT * FROM final_result
\gexec
```

### 4. Results Management
```vim
" Function to save query results
function! SaveQueryResults()
  let filename = input('Save results to: ', '', 'file')
  if filename != ''
    execute "write! " . filename
    echo "Results saved to " . filename
  endif
endfunction

nnoremap <leader>sr :call SaveQueryResults()<CR>

" Auto-format CSV results
autocmd BufRead,BufNewFile *.csv set filetype=csv
autocmd FileType csv TableModeEnable
```

### 5. Documentation Integration
```vim
" Generate query documentation
function! DocumentQuery()
  let purpose = input('Query purpose: ')
  let author = $USER
  let date = strftime('%Y-%m-%d')
  
  call append(line('.'), [
    \ '-- Purpose: ' . purpose,
    \ '-- Author: ' . author,
    \ '-- Date: ' . date,
    \ '-- Query:'
  \ ])
endfunction

command! DocQuery call DocumentQuery()
```

## Integration with Version Control

### Git Hooks for SQL Files
```bash
#!/bin/bash
# .git/hooks/pre-commit
# Validate SQL files before commit

for file in $(git diff --cached --name-only --diff-filter=ACM | grep '\.sql$'); do
  echo "Validating SQL file: $file"
  
  # Check for syntax errors (requires sqlfluff or similar)
  if command -v sqlfluff >/dev/null 2>&1; then
    sqlfluff lint "$file" || exit 1
  fi
  
  # Check for dangerous operations in production scripts
  if [[ $file == *"production"* ]] || [[ $file == *"prod"* ]]; then
    if grep -qi "drop\|delete\|truncate" "$file"; then
      echo "Warning: Destructive operation found in production script: $file"
      echo "Please review carefully."
    fi
  fi
done
```

### SQL File Templates
```bash
# ~/.vim/templates/sql_query.sql
-- =============================================================================
-- QUERY_NAME
-- =============================================================================
-- Description: 
-- Author: %USER%
-- Date: %DATE%
-- Database: 
-- Dependencies: 
-- =============================================================================

-- Parameters
\set param1 'default_value'

-- Main Query
WITH base_data AS (
  SELECT 
    -- columns
  FROM table_name
  WHERE condition
)
SELECT * 
FROM base_data
ORDER BY column
LIMIT 100;

-- Query Performance Notes:
-- Estimated execution time: 
-- Indexes required: 
-- Memory usage: 
```

This comprehensive guide provides everything needed to effectively integrate Vim with database development workflows, from basic psql integration to advanced plugin configurations and best practices.
