-- Create initial admin user
-- Username: admin
-- Password: admin123

INSERT INTO users (username, password_hash, email)
VALUES ('admin', '$2a$10$rN.qLwZxJYqF1bN8v8YTqupZmKxPYY5XYx7qwC8D8pxqMFNqP6Yri', 'admin@example.com')
ON CONFLICT (username) DO NOTHING;
