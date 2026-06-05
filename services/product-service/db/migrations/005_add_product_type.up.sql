CREATE TYPE product_type AS ENUM (
    'Electronics',
    'Mobile Phones & Accessories',
    'Fashion',
    'Home & Furniture',
    'Beauty & Health',
    'Appliances',
    'Automotive',
    'Hardware & Construction',
    'Agriculture',
    'Sports & Outdoors',
    'Baby & Kids',
    'Office Supplies',
    'Books & Education',
    'Pet Supplies',
    'Digital Products',
    'Services',
    'Liquids',
    'Beverages'
);

ALTER TABLE products ADD COLUMN product_type product_type NOT NULL DEFAULT 'Electronics';
