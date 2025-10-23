CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE subscriptions
(
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name TEXT NOT NULL,
    price        INT  NOT NULL CHECK (price >= 0),
    user_id      UUID NOT NULL,
    start_month  DATE NOT NULL,
    end_month    DATE,
    CHECK (end_month IS NULL OR end_month >= start_month)
);

CREATE INDEX idx_subscriptions_user ON subscriptions (user_id);
CREATE INDEX idx_subscriptions_service ON subscriptions (service_name);
CREATE INDEX idx_subscriptions_period ON subscriptions (start_month, end_month);
