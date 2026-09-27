DO $$
DECLARE
  new_org_id uuid;
BEGIN
  INSERT INTO organizations (id, name, email, phone)
  VALUES (gen_random_uuid(), 'Kanzanza', 'contact@kanzanza.com', '+243 000 000 000')
  RETURNING id INTO new_org_id;

  INSERT INTO users (id, email, password_hash, full_name, phone, role, organization_id, is_verified)
  VALUES (
    gen_random_uuid(),
    'admin@kanzanza.com',
    '$2a$10$7jDOE/Znqd7zyIPMqDK97.3Vht7iTCV6wWE6x.AM/awkpJKwbypwm',
    'Admin Kanzanza',
    '+243 000 000 000',
    'owner',
    new_org_id,
    true
  );
END $$;