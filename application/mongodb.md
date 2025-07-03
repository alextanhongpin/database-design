# MongoDB Integration

MongoDB integration patterns, best practices, and design considerations for NoSQL document database applications.

## 📚 Table of Contents

- [Overview](#overview)
- [Schema Design Patterns](#schema-design-patterns)
- [Referential Integrity](#referential-integrity)
- [Transaction Management](#transaction-management)
- [Query Patterns](#query-patterns)
- [Aggregation Framework](#aggregation-framework)
- [Performance Optimization](#performance-optimization)
- [Data Modeling Best Practices](#data-modeling-best-practices)
- [Common Patterns](#common-patterns)

## Overview

MongoDB is a NoSQL document database that stores data in flexible, JSON-like documents. Unlike relational databases, MongoDB doesn't enforce a rigid schema, allowing for more flexible data models.

### Key Differences from SQL Databases
- **Document-oriented**: Data stored as BSON documents
- **Schema flexibility**: Dynamic schema per document
- **Horizontal scaling**: Built-in sharding support
- **Embedded relationships**: Nest related data within documents
- **Rich query language**: Powerful aggregation framework

## Schema Design Patterns

### Embedding vs Referencing

#### Embedded Documents (One-to-Few)
```javascript
// ✅ Embed when data is frequently accessed together
const userWithAddresses = {
  _id: ObjectId("..."),
  name: "John Doe",
  email: "john@example.com",
  addresses: [
    {
      type: "home",
      street: "123 Main St",
      city: "Boston",
      country: "USA"
    },
    {
      type: "work",
      street: "456 Business Ave",
      city: "Boston",
      country: "USA"
    }
  ],
  createdAt: new Date()
};

// Query embedded data
db.users.find({ "addresses.type": "home" });
db.users.find({ "addresses.city": "Boston" });
```

#### Referenced Documents (One-to-Many)
```javascript
// ✅ Reference when child documents are large or numerous
const user = {
  _id: ObjectId("507f1f77bcf86cd799439011"),
  name: "John Doe",
  email: "john@example.com"
};

const post = {
  _id: ObjectId("507f1f77bcf86cd799439012"),
  title: "My Blog Post",
  content: "Lorem ipsum...",
  authorId: ObjectId("507f1f77bcf86cd799439011"),  // Reference to user
  tags: ["javascript", "mongodb"],
  publishedAt: new Date()
};

// Query with $lookup (MongoDB's join)
db.posts.aggregate([
  {
    $lookup: {
      from: "users",
      localField: "authorId",
      foreignField: "_id",
      as: "author"
    }
  },
  {
    $unwind: "$author"
  }
]);
```

### Polymorphic Pattern
```javascript
// Handle different document types in the same collection
const mediaCollection = [
  {
    _id: ObjectId("..."),
    type: "image",
    filename: "photo.jpg",
    dimensions: { width: 1920, height: 1080 },
    fileSize: 2048576
  },
  {
    _id: ObjectId("..."),
    type: "video",
    filename: "movie.mp4",
    duration: 180,
    resolution: "1080p",
    fileSize: 104857600
  },
  {
    _id: ObjectId("..."),
    type: "document",
    filename: "report.pdf",
    pageCount: 25,
    fileSize: 1048576
  }
];

// Query by type
db.media.find({ type: "image" });
db.media.find({ type: "video", duration: { $gte: 60 } });
```

## Referential Integrity

### Manual Referential Integrity
```javascript
// Since MongoDB doesn't enforce foreign keys, implement manually
class UserService {
  async deleteUser(userId) {
    const session = await mongoose.startSession();
    
    try {
      await session.withTransaction(async () => {
        // Check for dependencies
        const postCount = await Post.countDocuments({ authorId: userId });
        if (postCount > 0) {
          throw new Error(`Cannot delete user with ${postCount} posts`);
        }
        
        const commentCount = await Comment.countDocuments({ userId: userId });
        if (commentCount > 0) {
          throw new Error(`Cannot delete user with ${commentCount} comments`);
        }
        
        // Safe to delete
        await User.findByIdAndDelete(userId, { session });
        
        // Clean up any remaining references
        await Post.updateMany(
          { authorId: userId },
          { $unset: { authorId: 1 } },
          { session }
        );
      });
    } finally {
      await session.endSession();
    }
  }
}
```

### Cascade Delete Pattern
```javascript
// Implement cascade delete for parent-child relationships
async function deleteUserAndDependents(userId) {
  const session = await mongoose.startSession();
  
  try {
    await session.withTransaction(async () => {
      // Delete in dependency order
      await Comment.deleteMany({ userId }, { session });
      await Post.deleteMany({ authorId: userId }, { session });
      await UserProfile.deleteOne({ userId }, { session });
      await User.findByIdAndDelete(userId, { session });
    });
  } finally {
    await session.endSession();
  }
}
```

### Soft Delete Pattern
```javascript
// Implement soft delete to maintain referential integrity
const userSchema = new mongoose.Schema({
  name: String,
  email: String,
  deletedAt: { type: Date, default: null },
  isActive: { type: Boolean, default: true }
});

// Soft delete middleware
userSchema.methods.softDelete = function() {
  this.deletedAt = new Date();
  this.isActive = false;
  return this.save();
};

// Query only active users
userSchema.pre(/^find/, function() {
  this.where({ deletedAt: null, isActive: true });
});

// Usage
const user = await User.findById(userId);
await user.softDelete();  // Preserves referential integrity
```

## Transaction Management

### ACID Transactions (MongoDB 4.0+)
```javascript
// Multi-document transactions for ACID compliance
async function transferFunds(fromAccountId, toAccountId, amount) {
  const session = await mongoose.startSession();
  
  try {
    return await session.withTransaction(async () => {
      // Debit from source account
      const fromAccount = await Account.findByIdAndUpdate(
        fromAccountId,
        { $inc: { balance: -amount } },
        { session, new: true }
      );
      
      if (fromAccount.balance < 0) {
        throw new Error('Insufficient funds');
      }
      
      // Credit to destination account
      const toAccount = await Account.findByIdAndUpdate(
        toAccountId,
        { $inc: { balance: amount } },
        { session, new: true }
      );
      
      // Create transaction record
      await Transaction.create([{
        fromAccountId,
        toAccountId,
        amount,
        type: 'transfer',
        timestamp: new Date()
      }], { session });
      
      return { fromAccount, toAccount };
    });
  } finally {
    await session.endSession();
  }
}
```

### Two-Phase Commit Pattern
```javascript
// Manual two-phase commit for complex transactions
async function twoPhaseCommit(operations) {
  const transactionId = new ObjectId();
  
  try {
    // Phase 1: Prepare
    for (const operation of operations) {
      await operation.prepare(transactionId);
    }
    
    // Phase 2: Commit
    for (const operation of operations) {
      await operation.commit(transactionId);
    }
    
    // Cleanup
    await Transaction.deleteOne({ _id: transactionId });
    
  } catch (error) {
    // Rollback on any failure
    for (const operation of operations) {
      await operation.rollback(transactionId).catch(() => {});
    }
    throw error;
  }
}

class AccountOperation {
  constructor(accountId, amount) {
    this.accountId = accountId;
    this.amount = amount;
  }
  
  async prepare(transactionId) {
    await Account.updateOne(
      { _id: this.accountId },
      { 
        $push: { 
          pendingTransactions: { 
            id: transactionId, 
            amount: this.amount 
          } 
        } 
      }
    );
  }
  
  async commit(transactionId) {
    await Account.updateOne(
      { _id: this.accountId },
      {
        $inc: { balance: this.amount },
        $pull: { pendingTransactions: { id: transactionId } }
      }
    );
  }
  
  async rollback(transactionId) {
    await Account.updateOne(
      { _id: this.accountId },
      { $pull: { pendingTransactions: { id: transactionId } } }
    );
  }
}
```

## Query Patterns

### Efficient Queries
```javascript
// ✅ Use indexes effectively
db.users.createIndex({ email: 1 });
db.posts.createIndex({ authorId: 1, publishedAt: -1 });
db.posts.createIndex({ tags: 1 });  // Multikey index for arrays

// ✅ Project only needed fields
db.users.find(
  { status: "active" },
  { name: 1, email: 1, _id: 0 }  // Only return name and email
);

// ✅ Use explain to analyze query performance
db.posts.find({ authorId: ObjectId("...") }).explain("executionStats");
```

### Pagination Patterns
```javascript
// ✅ Cursor-based pagination (preferred for large datasets)
async function getPosts(lastId = null, limit = 10) {
  const query = lastId 
    ? { _id: { $gt: ObjectId(lastId) } }
    : {};
    
  return await db.posts
    .find(query)
    .sort({ _id: 1 })
    .limit(limit)
    .toArray();
}

// ✅ Skip-based pagination (for small datasets with page numbers)
async function getPostsPage(page = 1, limit = 10) {
  const skip = (page - 1) * limit;
  
  const [posts, total] = await Promise.all([
    db.posts.find().skip(skip).limit(limit).toArray(),
    db.posts.countDocuments()
  ]);
  
  return {
    posts,
    total,
    page,
    totalPages: Math.ceil(total / limit)
  };
}
```

### Text Search
```javascript
// Create text index
db.posts.createIndex({ 
  title: "text", 
  content: "text",
  tags: "text"
});

// Search with text index
db.posts.find({ 
  $text: { 
    $search: "mongodb database tutorial",
    $caseSensitive: false
  } 
});

// Search with scoring
db.posts.find(
  { $text: { $search: "mongodb tutorial" } },
  { score: { $meta: "textScore" } }
).sort({ score: { $meta: "textScore" } });
```

## Aggregation Framework

### Complex Aggregations
```javascript
// User activity dashboard
const userActivity = await db.posts.aggregate([
  // Match posts from last 30 days
  {
    $match: {
      publishedAt: {
        $gte: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)
      }
    }
  },
  
  // Group by author and calculate metrics
  {
    $group: {
      _id: "$authorId",
      postCount: { $sum: 1 },
      totalViews: { $sum: "$views" },
      avgViews: { $avg: "$views" },
      latestPost: { $max: "$publishedAt" },
      tags: { $addToSet: "$tags" }
    }
  },
  
  // Lookup author information
  {
    $lookup: {
      from: "users",
      localField: "_id",
      foreignField: "_id",
      as: "author"
    }
  },
  
  // Flatten author array
  { $unwind: "$author" },
  
  // Project final structure
  {
    $project: {
      authorName: "$author.name",
      authorEmail: "$author.email",
      postCount: 1,
      totalViews: 1,
      avgViews: { $round: ["$avgViews", 2] },
      latestPost: 1,
      uniqueTags: { $size: { $reduce: {
        input: "$tags",
        initialValue: [],
        in: { $setUnion: ["$$value", "$$this"] }
      }}}
    }
  },
  
  // Sort by total views descending
  { $sort: { totalViews: -1 } },
  
  // Limit to top 10
  { $limit: 10 }
]);
```

### Time-Series Aggregations
```javascript
// Daily active users over time
const dailyActiveUsers = await db.userSessions.aggregate([
  // Match last 30 days
  {
    $match: {
      startTime: {
        $gte: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)
      }
    }
  },
  
  // Group by date and count unique users
  {
    $group: {
      _id: {
        year: { $year: "$startTime" },
        month: { $month: "$startTime" },
        day: { $dayOfMonth: "$startTime" }
      },
      uniqueUsers: { $addToSet: "$userId" },
      totalSessions: { $sum: 1 }
    }
  },
  
  // Calculate final metrics
  {
    $project: {
      date: {
        $dateFromParts: {
          year: "$_id.year",
          month: "$_id.month",
          day: "$_id.day"
        }
      },
      activeUsers: { $size: "$uniqueUsers" },
      totalSessions: 1,
      avgSessionsPerUser: {
        $divide: ["$totalSessions", { $size: "$uniqueUsers" }]
      }
    }
  },
  
  { $sort: { date: 1 } }
]);
```

## Performance Optimization

### Index Strategies
```javascript
// Compound indexes for common query patterns
db.orders.createIndex({ 
  customerId: 1, 
  status: 1, 
  createdAt: -1 
});

// Partial indexes for sparse data
db.users.createIndex(
  { email: 1 },
  { 
    partialFilterExpression: { 
      email: { $exists: true, $ne: null } 
    } 
  }
);

// TTL indexes for automatic cleanup
db.sessions.createIndex(
  { createdAt: 1 },
  { expireAfterSeconds: 3600 }  // 1 hour
);

// Text indexes with weights
db.products.createIndex({
  name: "text",
  description: "text",
  tags: "text"
}, {
  weights: {
    name: 10,        // Name is most important
    tags: 5,         // Tags are moderately important
    description: 1   // Description is least important
  }
});
```

### Query Optimization
```javascript
// ❌ Inefficient: Multiple round trips
const posts = await Post.find({ authorId: userId });
for (const post of posts) {
  post.author = await User.findById(post.authorId);
  post.comments = await Comment.find({ postId: post._id });
}

// ✅ Efficient: Single aggregation pipeline
const postsWithDetails = await Post.aggregate([
  { $match: { authorId: ObjectId(userId) } },
  
  // Lookup author
  {
    $lookup: {
      from: "users",
      localField: "authorId", 
      foreignField: "_id",
      as: "author"
    }
  },
  
  // Lookup comments
  {
    $lookup: {
      from: "comments",
      localField: "_id",
      foreignField: "postId",
      as: "comments"
    }
  },
  
  { $unwind: "$author" },
  
  {
    $project: {
      title: 1,
      content: 1,
      publishedAt: 1,
      "author.name": 1,
      "author.email": 1,
      commentCount: { $size: "$comments" },
      recentComments: { $slice: ["$comments", -5] }
    }
  }
]);
```

### Connection Pooling
```javascript
// Mongoose connection optimization
const mongoose = require('mongoose');

const mongoOptions = {
  maxPoolSize: 10,          // Maximum connections
  minPoolSize: 2,           // Minimum connections
  maxIdleTimeMS: 30000,     // Close connections after 30s
  serverSelectionTimeoutMS: 5000,  // Timeout server selection
  socketTimeoutMS: 45000,   // Close sockets after 45s
  bufferCommands: false,    // Disable mongoose buffering
  bufferMaxEntries: 0       // Disable mongoose buffering
};

await mongoose.connect(mongoUri, mongoOptions);

// Monitor connection events
mongoose.connection.on('connected', () => {
  console.log('MongoDB connected');
});

mongoose.connection.on('error', (err) => {
  console.error('MongoDB error:', err);
});

mongoose.connection.on('disconnected', () => {
  console.log('MongoDB disconnected');
});
```

## Data Modeling Best Practices

### The 6 Rules of Thumb

1. **Favor embedding unless there is a compelling reason not to**
2. **Needing to access an object on its own is a compelling reason not to embed**
3. **Arrays should not grow without bound**
4. **Don't be afraid of application-level joins if they provide better query patterns**
5. **Consider the write/read ratio when denormalizing**
6. **How you model your data depends entirely on your particular application's data access patterns**

### Schema Validation
```javascript
// Define schema with validation
const userSchema = {
  $jsonSchema: {
    bsonType: "object",
    required: ["name", "email", "createdAt"],
    properties: {
      name: {
        bsonType: "string",
        minLength: 1,
        maxLength: 100,
        description: "must be a string between 1-100 characters"
      },
      email: {
        bsonType: "string",
        pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
        description: "must be a valid email address"
      },
      age: {
        bsonType: "int",
        minimum: 0,
        maximum: 150,
        description: "must be an integer between 0 and 150"
      },
      addresses: {
        bsonType: "array",
        maxItems: 5,
        items: {
          bsonType: "object",
          required: ["type", "street", "city"],
          properties: {
            type: {
              enum: ["home", "work", "other"]
            },
            street: { bsonType: "string" },
            city: { bsonType: "string" },
            country: { bsonType: "string" }
          }
        }
      },
      createdAt: {
        bsonType: "date"
      }
    }
  }
};

// Apply validation to collection
db.createCollection("users", {
  validator: userSchema
});
```

## Common Patterns

### Attribute Pattern
```javascript
// Instead of creating many sparse fields
const productBad = {
  name: "Laptop",
  price: 999,
  screenSize: "15 inch",      // Only for electronics
  material: null,             // Only for clothing
  color: "Silver",            // Common
  weight: "2.5 kg",          // Common
  batteryLife: "8 hours"      // Only for electronics
};

// Use attribute pattern for flexible schema
const productGood = {
  name: "Laptop",
  price: 999,
  category: "electronics",
  attributes: [
    { key: "screenSize", value: "15 inch", unit: "inch" },
    { key: "color", value: "Silver" },
    { key: "weight", value: 2.5, unit: "kg" },
    { key: "batteryLife", value: 8, unit: "hours" }
  ]
};

// Create index for efficient attribute queries
db.products.createIndex({ "attributes.key": 1, "attributes.value": 1 });

// Query attributes
db.products.find({ 
  "attributes": { 
    $elemMatch: { 
      key: "screenSize", 
      value: { $gte: "14 inch" } 
    } 
  } 
});
```

### Bucket Pattern
```javascript
// For time-series data, group documents into buckets
const temperatureReading = {
  _id: ObjectId("..."),
  deviceId: "sensor_001",
  year: 2023,
  month: 12,
  day: 15,
  hour: 14,  // Bucket by hour
  readings: [
    { minute: 0, temperature: 22.5, humidity: 65 },
    { minute: 1, temperature: 22.7, humidity: 64 },
    { minute: 2, temperature: 22.6, humidity: 65 },
    // ... up to 60 readings per hour
  ],
  readingCount: 60,
  avgTemperature: 22.6,
  maxTemperature: 23.1,
  minTemperature: 22.0
};

// Efficient queries on bucketed data
db.temperature.find({
  deviceId: "sensor_001",
  year: 2023,
  month: 12,
  day: 15,
  hour: { $gte: 10, $lte: 18 }
});
```

### Outlier Pattern
```javascript
// Handle documents that exceed normal size limits
const normalBook = {
  _id: ObjectId("..."),
  title: "Short Story",
  chapters: [
    { number: 1, title: "Beginning", content: "..." },
    { number: 2, title: "Middle", content: "..." },
    { number: 3, title: "End", content: "..." }
  ]
};

// For books with many chapters, use outlier pattern
const longBook = {
  _id: ObjectId("..."),
  title: "Encyclopedia",
  isOutlier: true,
  chapterCount: 500,
  chapters: [
    // Only store first few chapters inline
    { number: 1, title: "Introduction", content: "..." },
    { number: 2, title: "History", content: "..." }
  ]
};

// Store remaining chapters in separate collection
const extraChapters = {
  _id: ObjectId("..."),
  bookId: ObjectId("..."),  // Reference to main book
  chapters: [
    { number: 3, title: "Geography", content: "..." },
    { number: 4, title: "Culture", content: "..." },
    // ... many more chapters
  ]
};
```

## Error Handling

### Graceful Error Handling
```javascript
class MongoDBService {
  async createUser(userData) {
    try {
      const user = new User(userData);
      return await user.save();
    } catch (error) {
      if (error.code === 11000) {
        // Duplicate key error
        const field = Object.keys(error.keyPattern)[0];
        throw new Error(`${field} already exists`);
      }
      
      if (error.name === 'ValidationError') {
        const messages = Object.values(error.errors).map(e => e.message);
        throw new Error(`Validation failed: ${messages.join(', ')}`);
      }
      
      // Log unexpected errors
      console.error('Unexpected database error:', error);
      throw new Error('Database operation failed');
    }
  }
  
  async findUserWithRetry(id, maxRetries = 3) {
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        return await User.findById(id);
      } catch (error) {
        if (attempt === maxRetries) throw error;
        
        // Exponential backoff
        const delay = Math.pow(2, attempt) * 1000;
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
  }
}
```

## Useful Resources

### MongoDB Design Patterns
- [Building with Patterns](https://www.mongodb.com/blog/post/building-with-patterns-a-summary)
- [6 Rules of Thumb for MongoDB Schema Design](https://www.mongodb.com/blog/post/6-rules-of-thumb-for-mongodb-schema-design-part-1)

### Tools and Libraries
- **Mongoose**: ODM for Node.js with schema validation
- **MongoDB Compass**: GUI for database exploration
- **mongostat/mongotop**: Performance monitoring tools
- **MongoDB Charts**: Data visualization

### Best Practices Summary
1. **Design for your queries**: Model data based on access patterns
2. **Embed related data**: When accessed together frequently
3. **Reference large/independent data**: When accessed separately
4. **Use transactions judiciously**: For multi-document consistency
5. **Index strategically**: Based on query patterns
6. **Monitor performance**: Use explain() and profiling
7. **Handle errors gracefully**: Implement proper error handling
8. **Plan for scale**: Consider sharding and replication early
