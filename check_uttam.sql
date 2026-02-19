-- Check Uttam's SIP schedules
SELECT s.id, c.name, f.scheme_name, s.amount, s.start_date, s.is_active
FROM sip_schedules s
JOIN clients c ON s.client_id = c.id
JOIN funds f ON s.fund_id = f.id
WHERE c.name = 'Uttam';

-- Count total SIP schedules
SELECT COUNT(*) as total_sip_schedules FROM sip_schedules;

-- Check for duplicates
SELECT client_id, fund_id, amount, start_date, COUNT(*) as count
FROM sip_schedules
GROUP BY client_id, fund_id, amount, start_date
HAVING COUNT(*) > 1;
