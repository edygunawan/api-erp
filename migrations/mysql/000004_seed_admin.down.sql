-- Removes only the account reserved for this seed; assignments/sessions cascade.
DELETE FROM users WHERE id = '00000000-0000-4000-8002-000000000001';
