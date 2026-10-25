-- Bus Booking Platform - Purge Kanzanza agency activity
-- Deletes bookings, tickets, payments, AND journeys for agency "Kanzanza" while preserving
-- routes, buses, and the agency itself
-- Run this for targeted cleanup of Kanzanza test data

BEGIN;

-- 1. Supprimer uniquement les billets de l'agence Kanzanza
DELETE FROM tickets
WHERE journey_id IN (
    SELECT j.id FROM journeys j
    JOIN routes r ON j.route_id = r.id
    JOIN organizations o ON r.organization_id = o.id
    WHERE o.name ILIKE '%Kanzanza%'
);

-- 2. Supprimer uniquement les paiements de l'agence Kanzanza
-- Payments are linked via order_number which contains booking_reference
DELETE FROM payments
WHERE order_number IN (
    SELECT b.booking_reference FROM bookings b
    JOIN tickets t ON t.booking_id = b.id
    JOIN journeys j ON t.journey_id = j.id
    JOIN routes r ON j.route_id = r.id
    JOIN organizations o ON r.organization_id = o.id
    WHERE o.name ILIKE '%Kanzanza%'
);

-- 3. Supprimer uniquement les réservations de l'agence Kanzanza
-- (bookings that have tickets for Kanzanza journeys)
DELETE FROM bookings
WHERE id IN (
    SELECT DISTINCT t.booking_id FROM tickets t
    JOIN journeys j ON t.journey_id = j.id
    JOIN routes r ON j.route_id = r.id
    JOIN organizations o ON r.organization_id = o.id
    WHERE o.name ILIKE '%Kanzanza%'
);

-- 4. Supprimer les voyages (journeys) de l'agence Kanzanza
DELETE FROM journeys
WHERE route_id IN (
    SELECT r.id FROM routes r
    JOIN organizations o ON r.organization_id = o.id
    WHERE o.name ILIKE '%Kanzanza%'
);

COMMIT;