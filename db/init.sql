CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    monzo_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE modifiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(255) NOT NULL,
    value FLOAT NULL,
    percentage FLOAT NULL,
    include BOOLEAN DEFAULT FALSE
);

CREATE TABLE receipt_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item VARCHAR(255) NOT NULL,
    price FLOAT NULL,
    qty INTEGER NOT NULL
);

CREATE TABLE parsed_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    monzo_id VARCHAR(255) NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE parsed_receipts_items (
    parsed_receipt_id UUID NOT NULL REFERENCES parsed_receipts(id) ON DELETE CASCADE,
    receipt_item_id UUID NOT NULL REFERENCES receipt_items(id) ON DELETE CASCADE,
    PRIMARY KEY (parsed_receipt_id, receipt_item_id)
);

CREATE TABLE parsed_receipts_modifiers (
    parsed_receipt_id UUID NOT NULL REFERENCES parsed_receipts(id) ON DELETE CASCADE,
    modifier_id UUID NOT NULL REFERENCES modifiers(id) ON DELETE CASCADE,
    PRIMARY KEY (parsed_receipt_id, modifier_id)
);
