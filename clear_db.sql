-- Clear all data from database
DELETE FROM sip_schedules;
DELETE FROM transactions;
DELETE FROM nav_history;
DELETE FROM funds;
DELETE FROM clients;

-- Verify
SELECT 'Database cleared successfully!' AS status;
SELECT 'sip_schedules' AS table_name, COUNT(*) AS remaining_rows FROM sip_schedules
UNION ALL
SELECT 'transactions', COUNT(*) FROM transactions
UNION ALL
SELECT 'nav_history', COUNT(*) FROM nav_history
UNION ALL
SELECT 'funds', COUNT(*) FROM funds
UNION ALL
SELECT 'clients', COUNT(*) FROM clients;
