DROP TABLE IF EXISTS viewing_appointments;
DROP TABLE IF EXISTS sales_offers;
DROP TABLE IF EXISTS rental_bookings;
DROP TABLE IF EXISTS sales_listings_meta;
DROP TABLE IF EXISTS rental_listings_meta;
DROP TABLE IF EXISTS listings;
DROP TABLE IF EXISTS property_amenities;
DROP TABLE IF EXISTS property_images;
DROP TABLE IF EXISTS properties;

DROP TYPE IF EXISTS appointment_status;
DROP TYPE IF EXISTS offer_status;
DROP TYPE IF EXISTS booking_status;
DROP TYPE IF EXISTS listing_status;
DROP TYPE IF EXISTS listing_type;
DROP TYPE IF EXISTS property_type;

DROP EXTENSION IF EXISTS btree_gist;
DROP EXTENSION IF EXISTS postgis;
