-- Seed data for initial migration: users table.
INSERT INTO users (created_at, updated_at, username, password_hash)
VALUES (NOW(), NOW(), 'admin', '$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012');
