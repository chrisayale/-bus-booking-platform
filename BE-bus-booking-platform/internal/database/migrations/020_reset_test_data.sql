-- Bus Booking Platform - Reset test data migration
-- Deletes all bookings, tickets, payments, agents, and controllers for test phase reset
-- Run this to clean test data while preserving schema and reference data (organizations, owners, passengers, routes, buses, journeys)

-- Delete dependent data first (tickets reference bookings)
DELETE FROM tickets;

-- Delete payments (may reference bookings)
DELETE FROM payments;

-- Delete bookings
DELETE FROM bookings;

-- Delete agents and controllers (users with these roles)
DELETE FROM users WHERE role IN ('agent', 'controller');