-- Demo account so reviewers can log in without registering first — see
-- README.md, "Demo account". Password is "demo12345" (passwordMinLen is 8,
-- see internal/service/auth.go); hash generated with bcrypt.DefaultCost via
-- golang.org/x/crypto/bcrypt, the same package AuthService hashes with.
INSERT INTO users (id, email, name, password_hash)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'demo@example.local',
    'Demo User',
    '$2a$10$NBSH63El/OrXNMA4FnM.eusfcePZIjaV7kwy5.rD89mD/S3phaTwi'
);
