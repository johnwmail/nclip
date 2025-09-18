// MongoDB initialization script for nclip testing
db = db.getSiblingDB('nclip');

// Create TTL index for automatic expiration
db.pastes.createIndex(
  { "expires_at": 1 }, 
  { expireAfterSeconds: 0 }
);

print("TTL index created for automatic paste expiration");

// Create additional indexes for better performance during testing
db.pastes.createIndex({ "id": 1 }, { unique: true });
db.pastes.createIndex({ "created_at": 1 });

print("Additional indexes created for testing");