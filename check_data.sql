-- Check SIP schedules
SELECT 'SIP Schedules Count:' as info, COUNT(*) as count FROM sip_schedules;
SELECT client_id, COUNT(*) as schedule_count FROM sip_schedules GROUP BY client_id;

-- Check clients
SELECT 'Clients:' as info, id, name, pan FROM clients;

-- Check for duplicate SIP schedules
SELECT client_id, fund_id, amount, start_date, COUNT(*) as duplicates
FROM sip_schedules 
GROUP BY client_id, fund_id, amount, start_date
HAVING COUNT(*) > 1;
