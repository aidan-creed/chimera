-- This file defines temporary table schemas for sqlc's query generation.
-- it should NOT be included in database migration sequence

CREATE TABLE public.temp_items_staging (LIKE public.items INCLUDING DEFAULTS);
