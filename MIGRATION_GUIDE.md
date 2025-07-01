# Migration Guide: Topic-Based Reorganization

This guide helps you navigate the new topic-based organization of the Database Design Guide.

## 🔄 What Changed

The guide has been reorganized from a feature-based structure to a **topic-based structure** that better supports different learning paths and use cases.

### Old Structure → New Structure

```
database-design/
├── query/           → query-patterns/
├── patterns/        → (distributed across topics)
├── datatypes/       → data-types/
├── performance/     → performance/ (reorganized)
├── authorization/   → security/
├── administrative/  → operations/
├── client/          → application/
├── analytics/       → specialized/analytics/
├── temporal/        → specialized/time-series/
└── schema/          → examples/
```

## 📁 Detailed File Mapping

### Core Concepts
| Old Location | New Location | Topic |
|--------------|-------------|-------|
| `acid.md` | `fundamentals/acid-properties.md` | ACID Properties |
| `best-practices.md` | `fundamentals/design-principles.md` | Design Principles |
| `goals.md` | `fundamentals/design-goals.md` | Design Goals |
| `composite-type.md` | `fundamentals/composite-types.md` | Complex Types |

### Query Patterns
| Old Location | New Location | Topic |
|--------------|-------------|-------|
| `query/query.md` | `query-patterns/advanced/complex-queries.md` | Advanced Queries |
| `query/case.md` | `query-patterns/conditional/case-statements.md` | Conditional Logic |
| `query/update.md` | `query-patterns/manipulation/update-patterns.md` | Data Manipulation |
| `query/view.md` | `query-patterns/views/views.md` | Views |
| `query/materialized.md` | `query-patterns/views/materialized-views.md` | Materialized Views |
| `query/search.md` | `query-patterns/specialized/search.md` | Search Patterns |
| `query/soft-delete.md` | `query-patterns/specialized/soft-delete.md` | Soft Delete |
| `query/transaction.md` | `query-patterns/advanced/transactions.md` | Transactions |

### Schema Design
| Old Location | New Location | Topic |
|--------------|-------------|-------|
| `patterns/foreign-key.md` | `schema-design/foreign-keys.md` | Relationships |
| `patterns/constraint.md` | `schema-design/constraints.md` | Constraints |
| `patterns/polymorphic.md` | `schema-design/polymorphic.md` | Advanced Patterns |
| `patterns/inheritance.md` | `schema-design/inheritance.md` | Inheritance |

### Security & Access Control
| Old Location | New Location | Topic |
|--------------|-------------|-------|
| `authorization/` | `security/access-control/` | Access Control |
| `authorization/audit-logging.md` | `security/audit-logging.md` | Audit Logging |
| `authorization/role.md` | `security/roles.md` | Role Management |
| `authorization/row-level-security.md` | `security/row-level-security.md` | RLS |

### Operations & Administration
| Old Location | New Location | Topic |
|--------------|-------------|-------|
| `administrative/` | `operations/administration/` | Database Admin |
| `administrative/backup.md` | `operations/backup-recovery.md` | Backup & Recovery |
| `administrative/migration.md` | `operations/migrations.md` | Schema Migration |
| `incident/` | `operations/troubleshooting/` | Troubleshooting |

### Application Integration
| Old Location | New Location | Topic |
|--------------|-------------|-------|
| `client/` | `application/client-integration/` | Client Integration |
| `client/orm.md` | `application/orm-patterns.md` | ORM Patterns |
| `client/go.md` | `application/languages/go.md` | Language-Specific |
| `client/postgres.md` | `application/databases/postgresql.md` | Database-Specific |

### Specialized Topics
| Old Location | New Location | Topic |
|--------------|-------------|-------|
| `analytics/` | `specialized/analytics/` | Analytics |
| `temporal/` | `specialized/time-series/` | Time-Series Data |
| `patterns/payment.md` | `specialized/financial/payments.md` | Financial Systems |
| `patterns/multilingual.md` | `specialized/i18n/multilingual.md` | Internationalization |

## 🎯 Finding Content by Use Case

### "I want to learn database design from scratch"
**Start here**: `fundamentals/README.md`
- Follow the Beginner Learning Path
- Progress through: Fundamentals → Schema Design → Query Patterns

### "I need to optimize my queries"
**Go to**: `query-patterns/performance/`
- Query optimization techniques
- Index-friendly patterns
- Performance monitoring

### "I'm building an application with a database"
**Go to**: `application/README.md`
- ORM patterns and best practices
- Connection management
- Error handling strategies

### "I need to secure my database"
**Go to**: `security/README.md`
- Access control patterns
- Authentication strategies
- Audit logging implementation

### "I want to see real examples"
**Go to**: `examples/README.md`
- Complete schema examples
- Industry-specific patterns
- Case studies and solutions

## 🔗 Updated Cross-References

All internal links have been updated to reflect the new structure. However, if you have bookmarks or external references, update them as follows:

### Common URL Updates
```
# Old URLs
/query/case.md → /query-patterns/conditional/case-statements.md
/patterns/foreign-key.md → /schema-design/foreign-keys.md
/authorization/audit-logging.md → /security/audit-logging.md
/administrative/backup.md → /operations/backup-recovery.md
/client/orm.md → /application/orm-patterns.md
```

## 📚 New Learning Resources

### Topic-Specific READMEs
Each topic now has a comprehensive README with:
- Learning objectives
- Content organization
- Learning paths
- Cross-references
- Best practices checklists

### Enhanced Navigation
- **Topic-based browsing**: Find related content easily
- **Role-based paths**: Follow paths tailored to your role
- **Quick reference sections**: Access common patterns quickly
- **Cross-topic connections**: Understand how topics relate

## 🆘 Need Help Finding Something?

### Search Strategy
1. **Check the main README**: Look for your topic in the new organization
2. **Use topic READMEs**: Each topic has a detailed table of contents
3. **Follow learning paths**: Structured paths guide you through related content
4. **Check cross-references**: Links between related topics

### Still Can't Find It?
- Check the file mapping table above
- Look in the most logical topic area
- Search for keywords in the topic-specific READMEs
- The content is still there, just better organized!

## 🎉 Benefits of the New Organization

### For Learners
- **Clearer learning paths**: Progress logically through concepts
- **Better context**: Related information is grouped together
- **Reduced cognitive load**: Less searching, more learning

### For Practitioners
- **Task-oriented**: Find what you need for specific tasks
- **Role-based access**: Content organized by professional needs
- **Quick reference**: Common patterns are easily accessible

### For Contributors
- **Logical structure**: Easier to know where new content belongs
- **Better maintenance**: Related content stays together
- **Clearer scope**: Each topic has defined boundaries

---

The new organization maintains all existing content while making it more accessible and learnable. Start exploring with the [new main README](README_NEW.md)!
