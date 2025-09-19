// MongoDB initialization script for nclip
// This script runs when MongoDB container starts for the first time

print('Initializing nclip database...');

// Switch to nclip database
db = db.getSiblingDB('nclip');

// Create nclip user with read/write permissions on nclip database
db.createUser({
  user: 'nclip',
  pwd: 'secure_password_123',
  roles: [
    {
      role: 'readWrite',
      db: 'nclip'
    }
  ]
});

print('Created nclip user');

// Also create the same user in admin database for easier connection
db = db.getSiblingDB('admin');
db.createUser({
  user: 'nclip',
  pwd: 'secure_password_123',
  roles: [
    {
      role: 'readWrite',
      db: 'nclip'
    }
  ]
});

print('Created nclip user in admin database');

// Switch back to nclip database for collection setup
db = db.getSiblingDB('nclip');

// Create pastes collection
db.createCollection('pastes');

print('Created pastes collection');

// Create TTL index for automatic expiration
// This index will automatically remove documents when expires_at is reached
db.pastes.createIndex(
  { "expires_at": 1 },
  { 
    expireAfterSeconds: 0,
    name: "ttl_expires_at"
  }
);

print('Created TTL index on expires_at field');

// Create index on paste ID for faster lookups
db.pastes.createIndex(
  { "_id": 1 },
  { 
    name: "idx_paste_id",
    unique: true
  }
);

print('Created unique index on _id field');

// Create index on created_at for chronological queries
db.pastes.createIndex(
  { "created_at": -1 },
  { 
    name: "idx_created_at"
  }
);

print('Created index on created_at field');

// Create compound index for burn-after-read queries
db.pastes.createIndex(
  { 
    "burn_after_read": 1,
    "read_count": 1
  },
  { 
    name: "idx_burn_after_read"
  }
);

print('Created compound index for burn-after-read functionality');

// Verify collections and indexes
print('Collections in nclip database:');
db.getCollectionNames().forEach(function(collection) {
  print('  - ' + collection);
});

print('Indexes on pastes collection:');
db.pastes.getIndexes().forEach(function(index) {
  print('  - ' + index.name + ': ' + JSON.stringify(index.key));
});

print('MongoDB initialization completed successfully!');