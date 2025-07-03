# ORM (Object-Relational Mapping)

Comprehensive guide to Object-Relational Mapping patterns, best practices, and when to use (or avoid) ORMs in application development.

## 📚 Table of Contents

- [Overview](#overview)
- [When to Use ORMs](#when-to-use-orms)
- [When to Avoid ORMs](#when-to-avoid-orms)
- [ORM Patterns](#orm-patterns)
- [Popular ORMs by Language](#popular-orms-by-language)
- [Performance Considerations](#performance-considerations)
- [Best Practices](#best-practices)
- [Alternatives to ORMs](#alternatives-to-orms)
- [Migration Strategies](#migration-strategies)

## Overview

The fundamental question isn't "should I use an ORM?" but rather "how can I maximize my SQL capabilities while maintaining productivity?" ORMs provide abstraction over database operations, but this abstraction comes with trade-offs.

### ORM Benefits
- **Rapid Development**: Less boilerplate code for CRUD operations
- **Type Safety**: Compile-time checking for database operations
- **Database Abstraction**: Theoretical database portability
- **Developer Experience**: IDE support, autocompletion, refactoring
- **Relationship Management**: Automatic handling of foreign keys and joins
- **Security**: Built-in protection against SQL injection

### ORM Drawbacks
- **Performance Overhead**: Additional abstraction layer
- **Limited Database Features**: Can't leverage database-specific optimizations
- **Complex Queries**: Poor support for advanced SQL features
- **Hidden Complexity**: Generated queries may be inefficient
- **Learning Curve**: Need to understand both ORM and underlying SQL
- **Vendor Lock-in**: Tied to specific ORM patterns and conventions

## When to Use ORMs

### ✅ Ideal Use Cases

#### CRUD-Heavy Applications
```javascript
// Express/TypeORM - Simple CRUD operations
@Entity()
export class User {
  @PrimaryGeneratedColumn()
  id: number;

  @Column()
  name: string;

  @Column({ unique: true })
  email: string;

  @CreateDateColumn()
  createdAt: Date;

  @UpdateDateColumn()
  updatedAt: Date;
}

@Injectable()
export class UserService {
  constructor(
    @InjectRepository(User)
    private userRepository: Repository<User>
  ) {}

  async create(userData: CreateUserDto): Promise<User> {
    const user = this.userRepository.create(userData);
    return this.userRepository.save(user);
  }

  async findAll(): Promise<User[]> {
    return this.userRepository.find();
  }

  async findOne(id: number): Promise<User> {
    return this.userRepository.findOne({ where: { id } });
  }

  async update(id: number, updateData: UpdateUserDto): Promise<User> {
    await this.userRepository.update(id, updateData);
    return this.findOne(id);
  }

  async remove(id: number): Promise<void> {
    await this.userRepository.delete(id);
  }
}
```

#### Rapid Prototyping
```python
# Django models for quick development
from django.db import models
from django.contrib.auth.models import User

class Post(models.Model):
    title = models.CharField(max_length=200)
    content = models.TextField()
    author = models.ForeignKey(User, on_delete=models.CASCADE)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)
    is_published = models.BooleanField(default=False)

    class Meta:
        ordering = ['-created_at']

    def __str__(self):
        return self.title

# Automatic admin interface
@admin.register(Post)
class PostAdmin(admin.ModelAdmin):
    list_display = ['title', 'author', 'created_at', 'is_published']
    list_filter = ['is_published', 'created_at']
    search_fields = ['title', 'content']
```

#### Team with Limited SQL Experience
```java
// Spring Data JPA - Declarative queries
@Entity
@Table(name = "products")
public class Product {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    
    @Column(nullable = false)
    private String name;
    
    @Column(precision = 10, scale = 2)
    private BigDecimal price;
    
    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "category_id")
    private Category category;
    
    // getters/setters
}

@Repository
public interface ProductRepository extends JpaRepository<Product, Long> {
    List<Product> findByNameContaining(String name);
    List<Product> findByCategoryIdAndPriceBetween(Long categoryId, BigDecimal minPrice, BigDecimal maxPrice);
    
    @Query("SELECT p FROM Product p WHERE p.category.name = :categoryName ORDER BY p.price DESC")
    List<Product> findByCategoryNameOrderByPriceDesc(@Param("categoryName") String categoryName);
}
```

## When to Avoid ORMs

### ❌ Problematic Use Cases

#### Analytics and Reporting
```sql
-- Complex analytics query that's difficult to express in ORM
WITH monthly_sales AS (
  SELECT 
    DATE_TRUNC('month', order_date) as month,
    SUM(total_amount) as revenue,
    COUNT(*) as order_count,
    COUNT(DISTINCT customer_id) as unique_customers
  FROM orders 
  WHERE order_date >= '2023-01-01'
  GROUP BY DATE_TRUNC('month', order_date)
),
growth_rates AS (
  SELECT 
    month,
    revenue,
    LAG(revenue) OVER (ORDER BY month) as prev_revenue,
    ROUND(
      ((revenue - LAG(revenue) OVER (ORDER BY month)) / 
       LAG(revenue) OVER (ORDER BY month) * 100), 2
    ) as growth_rate
  FROM monthly_sales
)
SELECT 
  month,
  revenue,
  growth_rate,
  CASE 
    WHEN growth_rate > 10 THEN 'High Growth'
    WHEN growth_rate > 0 THEN 'Positive Growth'
    WHEN growth_rate < 0 THEN 'Decline'
    ELSE 'Stable'
  END as performance_category
FROM growth_rates
ORDER BY month;
```

#### High-Performance Applications
```go
// Direct SQL for performance-critical operations
func (r *OrderRepository) GetTopCustomersByRevenue(ctx context.Context, limit int) ([]CustomerRevenue, error) {
    query := `
        SELECT 
            c.id,
            c.name,
            c.email,
            SUM(o.total_amount) as total_revenue,
            COUNT(o.id) as order_count,
            AVG(o.total_amount) as avg_order_value,
            MAX(o.created_at) as last_order_date
        FROM customers c
        INNER JOIN orders o ON c.id = o.customer_id
        WHERE o.status = 'completed'
          AND o.created_at >= NOW() - INTERVAL '1 year'
        GROUP BY c.id, c.name, c.email
        HAVING SUM(o.total_amount) > 1000
        ORDER BY total_revenue DESC
        LIMIT $1
    `
    
    rows, err := r.db.QueryContext(ctx, query, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to query top customers: %w", err)
    }
    defer rows.Close()
    
    var customers []CustomerRevenue
    for rows.Next() {
        var customer CustomerRevenue
        err := rows.Scan(
            &customer.ID,
            &customer.Name,
            &customer.Email,
            &customer.TotalRevenue,
            &customer.OrderCount,
            &customer.AvgOrderValue,
            &customer.LastOrderDate,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan customer: %w", err)
        }
        customers = append(customers, customer)
    }
    
    return customers, nil
}
```

#### Database-Specific Features
```sql
-- PostgreSQL-specific features hard to use with ORMs
CREATE OR REPLACE FUNCTION update_user_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.bio, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.skills, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_user_search_trigger
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_user_search_vector();

-- Full-text search with ranking
SELECT 
    users.*,
    ts_rank(search_vector, plainto_tsquery('english', $1)) as rank
FROM users
WHERE search_vector @@ plainto_tsquery('english', $1)
ORDER BY rank DESC;
```

## Quick Reference

### When to Choose ORM
- ✅ Rapid prototyping and development
- ✅ Team has limited SQL experience
- ✅ CRUD-heavy applications
- ✅ Need type safety and IDE support
- ✅ Database portability is important

### When to Avoid ORM
- ❌ Performance-critical applications
- ❌ Complex analytics and reporting
- ❌ Heavy use of database-specific features
- ❌ Team has strong SQL expertise
- ❌ Need fine-grained query control

### ORM Performance Tips
1. **Profile generated queries** - Always check what SQL is generated
2. **Use eager loading** - Avoid N+1 query problems
3. **Implement strategic indexing** - Based on query patterns
4. **Connection pooling** - Configure appropriate pool sizes
5. **Hybrid approach** - Use ORMs for CRUD, raw SQL for complex operations

### Popular ORM Choices
- **JavaScript**: Prisma, TypeORM, Sequelize
- **Python**: SQLAlchemy, Django ORM
- **Java**: Hibernate/JPA, MyBatis
- **C#**: Entity Framework Core
- **Go**: GORM, Ent
- **Ruby**: Active Record
- **PHP**: Eloquent, Doctrine

Remember: The best approach often combines ORMs for productivity with raw SQL for performance-critical operations.
