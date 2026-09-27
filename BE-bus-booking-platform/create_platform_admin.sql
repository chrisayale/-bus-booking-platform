INSERT INTO users (id, email, password_hash, full_name, phone, role, is_verified, approval_status)
VALUES (
  gen_random_uuid(),
  'superadmin@ayaletrans.com',
  '$2a$10$7jDOE/Znqd7zyIPMqDK97.3Vht7iTCV6wWE6x.AM/awkpJKwbypwm',
  'Super Admin',
  '+243000000000',
  'platform_admin',
  true,
  'approved'
);