DROP TABLE IF EXISTS tracking_logs;
DROP TABLE IF EXISTS station_packages;
DROP TABLE IF EXISTS courier_tasks;
DROP TABLE IF EXISTS delivery_orders;
DROP TABLE IF EXISTS stations;
DROP TABLE IF EXISTS couriers;
DROP TABLE IF EXISTS logistics_partners;

DROP TYPE IF EXISTS station_package_direction;
DROP TYPE IF EXISTS station_status;
DROP TYPE IF EXISTS courier_status;
DROP TYPE IF EXISTS carrier_type;
DROP TYPE IF EXISTS delivery_type;
DROP TYPE IF EXISTS delivery_status;

DROP EXTENSION IF EXISTS postgis;
