-- Some upcoming bookings are public, so «For you» has some to join: every
-- other one at a facility with room for more than the owner (the local
-- tester's own, inserted later, stays private).
UPDATE booking b
SET is_public = TRUE
FROM (
    SELECT b2.id, row_number() OVER (ORDER BY b2.starts_at) AS n
    FROM booking b2
    JOIN facility f ON f.id = b2.facility_id
    WHERE b2.status = 'confirmed' AND b2.starts_at > NOW() AND f.capacity > 1 AND b2.user_id IS NOT NULL
) AS pick
WHERE b.id = pick.id AND pick.n % 2 = 1;
