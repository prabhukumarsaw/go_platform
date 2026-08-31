-- 013_focus_heartland_editions.up.sql
-- Comprehensive district coverage and regional seed articles for priority focal states:
-- Jharkhand, Bihar, Odisha, West Bengal (Kolkata), Uttar Pradesh, Delhi (NCR), Chhattisgarh

-- ──────────────────────────────────────────────
-- 1. DISTRICT SEEDING FOR PRIORITY STATES
-- ──────────────────────────────────────────────
DO $$
DECLARE
    t_jh INT;
    t_br INT;
    t_or INT;
    t_wb INT;
    t_up INT;
    t_dl INT;
    t_cg INT;
    author_id BIGINT;
    s_id UUID;
BEGIN
    SELECT id INTO t_jh FROM tenants WHERE slug = 'jharkhand';
    SELECT id INTO t_br FROM tenants WHERE slug = 'bihar';
    SELECT id INTO t_or FROM tenants WHERE slug = 'odisha';
    SELECT id INTO t_wb FROM tenants WHERE slug = 'west-bengal';
    SELECT id INTO t_up FROM tenants WHERE slug = 'uttar-pradesh';
    SELECT id INTO t_dl FROM tenants WHERE slug = 'delhi';
    SELECT id INTO t_cg FROM tenants WHERE slug = 'chhattisgarh';
    
    SELECT id INTO author_id FROM users WHERE email = 'superadmin@newsplatform.in' LIMIT 1;
    IF author_id IS NULL THEN
        SELECT id INTO author_id FROM users LIMIT 1;
    END IF;

    -- ────────────────── JHARKHAND (24 Districts) ──────────────────
    IF t_jh IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_jh, 'Ranchi', 'ranchi'),
            (t_jh, 'East Singhbhum (Jamshedpur)', 'jamshedpur'),
            (t_jh, 'Dhanbad', 'dhanbad'),
            (t_jh, 'Bokaro', 'bokaro'),
            (t_jh, 'Deoghar', 'deoghar'),
            (t_jh, 'Hazaribagh', 'hazaribagh'),
            (t_jh, 'Dumka', 'dumka'),
            (t_jh, 'Giridih', 'giridih'),
            (t_jh, 'Palamu', 'palamu'),
            (t_jh, 'Ramgarh', 'ramgarh'),
            (t_jh, 'West Singhbhum (Chaibasa)', 'chaibasa'),
            (t_jh, 'Godda', 'godda'),
            (t_jh, 'Sahebganj', 'sahebganj'),
            (t_jh, 'Koderma', 'koderma'),
            (t_jh, 'Gumla', 'gumla'),
            (t_jh, 'Latehar', 'latehar'),
            (t_jh, 'Simdega', 'simdega'),
            (t_jh, 'Lohardaga', 'lohardaga'),
            (t_jh, 'Jamtara', 'jamtara'),
            (t_jh, 'Pakur', 'pakur'),
            (t_jh, 'Khunti', 'khunti'),
            (t_jh, 'Garhwa', 'garhwa'),
            (t_jh, 'Chatra', 'chatra'),
            (t_jh, 'Saraikela-Kharsawan', 'saraikela')
        ON CONFLICT (tenant_id, slug) DO NOTHING;

        -- Seed Story & Article for Jharkhand
        INSERT INTO stories (tenant_id, slug) VALUES (t_jh, 'ranchi-industrial-corridor') RETURNING id INTO s_id;

        INSERT INTO articles (story_id, tenant_id, author_id, title, slug, excerpt, body, language, status, is_breaking, is_featured, published_at)
        VALUES (
            s_id, t_jh, author_id,
            'Ranchi Ring Road & Smart Industrial Corridor Project Approved for Faster Mineral Freight Logistics',
            'ranchi-ring-road-smart-industrial-corridor-approved',
            'The state government has sanctioned capital outlays to decongest mining corridors and accelerate connectivity between Ranchi, Jamshedpur, and Dhanbad.',
            '[{"type":"paragraph","text":"The state cabinet today approved a multi-lane heavy freight corridor bypassing dense urban zones in Ranchi and connecting key manufacturing clusters in Jamshedpur and Bokaro."},{"type":"fact_check","claim":"Project timeline set for completion by Q4 2027.","explanation":"Cabinet documents confirm phase 1 civil contracts have been tendered with strict environmental clearance protocols.","verdict":"VERIFIED"}]'::jsonb,
            'en', 'published', TRUE, TRUE, NOW()
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

    -- ────────────────── BIHAR (38 Districts) ──────────────────
    IF t_br IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_br, 'Patna', 'patna'),
            (t_br, 'Gaya', 'gaya'),
            (t_br, 'Muzaffarpur', 'muzaffarpur'),
            (t_br, 'Bhagalpur', 'bhagalpur'),
            (t_br, 'Darbhanga', 'darbhanga'),
            (t_br, 'Purnia', 'purnia'),
            (t_br, 'Begusarai', 'begusarai'),
            (t_br, 'Nalanda (Bihar Sharif)', 'nalanda'),
            (t_br, 'Saran (Chhapra)', 'chhapra'),
            (t_br, 'Rohtas (Sasaram)', 'sasaram'),
            (t_br, 'Samastipur', 'samastipur'),
            (t_br, 'Vaishali (Hajipur)', 'hajipur'),
            (t_br, 'Siwan', 'siwan'),
            (t_br, 'Katihar', 'katihar'),
            (t_br, 'Munger', 'munger'),
            (t_br, 'Madhubani', 'madhubani'),
            (t_br, 'West Champaran (Bettiah)', 'bettiah'),
            (t_br, 'East Champaran (Motihari)', 'motihari'),
            (t_br, 'Bhojpur (Arrah)', 'arrah'),
            (t_br, 'Nawada', 'nawada'),
            (t_br, 'Buxar', 'buxar'),
            (t_br, 'Gopalganj', 'gopalganj'),
            (t_br, 'Sitamarhi', 'sitamarhi'),
            (t_br, 'Aurangabad', 'aurangabad-bihar'),
            (t_br, 'Saharsa', 'saharsa'),
            (t_br, 'Supaul', 'supaul'),
            (t_br, 'Jamui', 'jamui'),
            (t_br, 'Madhepura', 'madhepura'),
            (t_br, 'Khagaria', 'khagaria'),
            (t_br, 'Banka', 'banka'),
            (t_br, 'Kishanganj', 'kishanganj'),
            (t_br, 'Kaimur (Bhabua)', 'kaimur'),
            (t_br, 'Arwal', 'arwal'),
            (t_br, 'Lakhisarai', 'lakhisarai'),
            (t_br, 'Jehanabad', 'jehanabad'),
            (t_br, 'Sheikhpura', 'sheikhpura'),
            (t_br, 'Sheohar', 'sheohar')
        ON CONFLICT (tenant_id, slug) DO NOTHING;

        -- Seed Story & Article for Bihar
        INSERT INTO stories (tenant_id, slug) VALUES (t_br, 'patna-metro-priority-corridor') RETURNING id INTO s_id;

        INSERT INTO articles (story_id, tenant_id, author_id, title, slug, excerpt, body, language, status, is_breaking, is_featured, published_at)
        VALUES (
            s_id, t_br, author_id,
            'Patna Metro Priority Corridor Completes Underground Tunneling Ahead of Schedule',
            'patna-metro-priority-corridor-underground-tunneling-completed',
            'Engineers have achieved major milestones connecting Patna Junction to Rajendra Nagar Terminal, with commercial trial runs scheduled for late 2026.',
            '[{"type":"paragraph","text":"Patna Metro Rail Corporation announced the completion of double-shield tunnel boring operations across the dense historic corridor between Patna Junction and Gandhi Maidan."},{"type":"quote","quote":"This priority stretch will transform urban transit and cut inner-city commute times by over 60 percent.","author":"Chief Project Director","designation":"PMRCL"}]'::jsonb,
            'en', 'published', FALSE, TRUE, NOW()
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

    -- ────────────────── ODISHA (30 Districts) ──────────────────
    IF t_or IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_or, 'Khordha (Bhubaneswar)', 'khordha-bhubaneswar'),
            (t_or, 'Cuttack', 'cuttack'),
            (t_or, 'Sundargarh (Rourkela)', 'rourkela'),
            (t_or, 'Puri', 'puri'),
            (t_or, 'Sambalpur', 'sambalpur'),
            (t_or, 'Ganjam (Berhampur)', 'berhampur'),
            (t_or, 'Balasore', 'balasore'),
            (t_or, 'Mayurbhanj (Baripada)', 'baripada'),
            (t_or, 'Angul', 'angul'),
            (t_or, 'Jharsuguda', 'jharsuguda'),
            (t_or, 'Balangir', 'balangir'),
            (t_or, 'Bargarh', 'bargarh'),
            (t_or, 'Bhadrak', 'bhadrak'),
            (t_or, 'Jajpur', 'jajpur'),
            (t_or, 'Kendrapara', 'kendrapara'),
            (t_or, 'Keonjhar', 'keonjhar'),
            (t_or, 'Koraput', 'koraput'),
            (t_or, 'Rayagada', 'rayagada'),
            (t_or, 'Dhenkanal', 'dhenkanal'),
            (t_or, 'Jagatsinghpur', 'jagatsinghpur'),
            (t_or, 'Kalahandi', 'kalahandi'),
            (t_or, 'Kandhamal', 'kandhamal'),
            (t_or, 'Malkangiri', 'malkangiri'),
            (t_or, 'Nabarangpur', 'nabarangpur'),
            (t_or, 'Nuapada', 'nuapada'),
            (t_or, 'Nayagarh', 'nayagarh'),
            (t_or, 'Subarnapur', 'subarnapur'),
            (t_or, 'Deogarh', 'deogarh-odisha'),
            (t_or, 'Gajapati', 'gajapati'),
            (t_or, 'Boudh', 'boudh')
        ON CONFLICT (tenant_id, slug) DO NOTHING;

        -- Seed Story & Article for Odisha
        INSERT INTO stories (tenant_id, slug) VALUES (t_or, 'bhubaneswar-semiconductor-hub') RETURNING id INTO s_id;

        INSERT INTO articles (story_id, tenant_id, author_id, title, slug, excerpt, body, language, status, is_breaking, is_featured, published_at)
        VALUES (
            s_id, t_or, author_id,
            'Bhubaneswar Emerges as Key Semiconductor and Green Energy Innovation Capital',
            'bhubaneswar-semiconductor-green-energy-innovation-hub',
            'New fabrication and silicon design testbeds inaugurated near Infocity, attracting major global tech investments to the temple city.',
            '[{"type":"paragraph","text":"Leading multinational electronics manufacturers have signed MoUs to set up fabless design centers and silicon assembly units in Bhubaneswar."},{"type":"callout","title":"Investment Inflow","text":"Over ₹12,500 Crores in commitments confirmed across semiconductor packaging and aerospace testing."}]'::jsonb,
            'en', 'published', TRUE, TRUE, NOW()
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

    -- ────────────────── WEST BENGAL / KOLKATA (23 Districts) ──────────────────
    IF t_wb IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_wb, 'Kolkata', 'kolkata'),
            (t_wb, 'Howrah', 'howrah'),
            (t_wb, 'North 24 Parganas', 'north-24-parganas'),
            (t_wb, 'South 24 Parganas', 'south-24-parganas'),
            (t_wb, 'Hooghly', 'hooghly'),
            (t_wb, 'Paschim Bardhaman (Asansol/Durgapur)', 'asansol-durgapur'),
            (t_wb, 'Purba Bardhaman', 'bardhaman'),
            (t_wb, 'Paschim Medinipur', 'medinipur'),
            (t_wb, 'Purba Medinipur', 'tamluk'),
            (t_wb, 'Darjeeling', 'darjeeling'),
            (t_wb, 'Siliguri', 'siliguri'),
            (t_wb, 'Jalpaiguri', 'jalpaiguri'),
            (t_wb, 'Murshidabad', 'murshidabad'),
            (t_wb, 'Nadia', 'nadia'),
            (t_wb, 'Malda', 'malda'),
            (t_wb, 'Birbhum', 'birbhum'),
            (t_wb, 'Bankura', 'bankura'),
            (t_wb, 'Purulia', 'purulia'),
            (t_wb, 'Cooch Behar', 'cooch-behar'),
            (t_wb, 'Alipurduar', 'alipurduar'),
            (t_wb, 'Uttar Dinajpur', 'uttar-dinajpur'),
            (t_wb, 'Dakshin Dinajpur', 'dakshin-dinajpur'),
            (t_wb, 'Kalimpong', 'kalimpong')
        ON CONFLICT (tenant_id, slug) DO NOTHING;

        -- Seed Story & Article for West Bengal
        INSERT INTO stories (tenant_id, slug) VALUES (t_wb, 'kolkata-underwater-metro') RETURNING id INTO s_id;

        INSERT INTO articles (story_id, tenant_id, author_id, title, slug, excerpt, body, language, status, is_breaking, is_featured, published_at)
        VALUES (
            s_id, t_wb, author_id,
            'Kolkata Underwater Metro East-West Expansion Reaches Record Daily Commuter Traffic',
            'kolkata-underwater-metro-east-west-ridership-record',
            'The riverine transit link under the Hooghly river logs over 350,000 passenger trips daily, significantly easing Howrah Bridge congestion.',
            '[{"type":"paragraph","text":"Kolkata Metro Rail Corporation (KMRC) confirmed that cross-river passenger journeys between Howrah and Esplanade have cut cross-city commute time to under 8 minutes."}]'::jsonb,
            'en', 'published', FALSE, TRUE, NOW()
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

    -- ────────────────── UTTAR PRADESH (75 Districts Key Hubs) ──────────────────
    IF t_up IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_up, 'Lucknow', 'lucknow'),
            (t_up, 'Varanasi', 'varanasi'),
            (t_up, 'Kanpur Nagar', 'kanpur'),
            (t_up, 'Prayagraj', 'prayagraj'),
            (t_up, 'Gautam Buddha Nagar (Noida)', 'noida'),
            (t_up, 'Ghaziabad', 'ghaziabad'),
            (t_up, 'Agra', 'agra'),
            (t_up, 'Gorakhpur', 'gorakhpur'),
            (t_up, 'Ayodhya', 'ayodhya'),
            (t_up, 'Meerut', 'meerut'),
            (t_up, 'Bareilly', 'bareilly'),
            (t_up, 'Aligarh', 'aligarh'),
            (t_up, 'Moradabad', 'moradabad'),
            (t_up, 'Jhansi', 'jhansi'),
            (t_up, 'Mathura', 'mathura'),
            (t_up, 'Saharanpur', 'saharanpur'),
            (t_up, 'Muzaffarnagar', 'muzaffarnagar'),
            (t_up, 'Firozabad', 'firozabad'),
            (t_up, 'Mirzapur', 'mirzapur'),
            (t_up, 'Azamgarh', 'azamgarh')
        ON CONFLICT (tenant_id, slug) DO NOTHING;

        -- Seed Story & Article for Uttar Pradesh
        INSERT INTO stories (tenant_id, slug) VALUES (t_up, 'noida-jewar-airport') RETURNING id INTO s_id;

        INSERT INTO articles (story_id, tenant_id, author_id, title, slug, excerpt, body, language, status, is_breaking, is_featured, published_at)
        VALUES (
            s_id, t_up, author_id,
            'Noida International Airport Jewar Finalizes Commercial Flight Calibrations Ahead of Inauguration',
            'noida-international-airport-jewar-flight-calibrations-completed',
            'DGCA and AAI complete multi-aircraft ILS flight checks, preparing western UP for one of Asia’s largest aviation hubs.',
            '[{"type":"paragraph","text":"Runway calibrations and baggage handling trial operations at Jewar Airport were successfully concluded this morning under supervision of civil aviation inspectors."},{"type":"fact_check","claim":"Direct expressway link between Delhi, Noida, and Jewar operational.","explanation":"The dedicated 8-lane expressway spur connecting Yamuna Expressway and the airport terminal is completed.","verdict":"VERIFIED"}]'::jsonb,
            'en', 'published', TRUE, TRUE, NOW()
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

    -- ────────────────── DELHI (NCR) ──────────────────
    IF t_dl IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_dl, 'New Delhi', 'new-delhi'),
            (t_dl, 'Central Delhi', 'central-delhi'),
            (t_dl, 'South Delhi', 'south-delhi'),
            (t_dl, 'North Delhi', 'north-delhi'),
            (t_dl, 'East Delhi', 'east-delhi'),
            (t_dl, 'West Delhi', 'west-delhi'),
            (t_dl, 'South West Delhi', 'south-west-delhi'),
            (t_dl, 'South East Delhi', 'south-east-delhi'),
            (t_dl, 'North West Delhi', 'north-west-delhi'),
            (t_dl, 'North East Delhi', 'north-east-delhi'),
            (t_dl, 'Shahdara', 'shahdara')
        ON CONFLICT (tenant_id, slug) DO NOTHING;

        -- Seed Story & Article for Delhi
        INSERT INTO stories (tenant_id, slug) VALUES (t_dl, 'delhi-electric-bus-fleet') RETURNING id INTO s_id;

        INSERT INTO articles (story_id, tenant_id, author_id, title, slug, excerpt, body, language, status, is_breaking, is_featured, published_at)
        VALUES (
            s_id, t_dl, author_id,
            'Delhi NCR Expands Zero-Emission Electric Bus Fleet to Over 3,500 Units',
            'delhi-ncr-zero-emission-electric-bus-fleet-expansion',
            'The capital marks a major air quality and urban transit transition with state-of-the-art low-floor electric transit coaches.',
            '[{"type":"paragraph","text":"Transport authorities rolled out 400 additional low-floor air-conditioned electric buses across outer and central ring routes today."}]'::jsonb,
            'en', 'published', TRUE, TRUE, NOW()
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

    -- ────────────────── CHHATTISGARH ──────────────────
    IF t_cg IS NOT NULL THEN
        INSERT INTO districts (tenant_id, name, slug) VALUES
            (t_cg, 'Raipur', 'raipur'),
            (t_cg, 'Bilaspur', 'bilaspur'),
            (t_cg, 'Durg', 'durg'),
            (t_cg, 'Bhilai', 'bhilai'),
            (t_cg, 'Bastar (Jagdalpur)', 'jagdalpur'),
            (t_cg, 'Korba', 'korba'),
            (t_cg, 'Rajnandgaon', 'rajnandgaon'),
            (t_cg, 'Raigarh', 'raigarh'),
            (t_cg, 'Surguja (Ambikapur)', 'ambikapur'),
            (t_cg, 'Janjgir-Champa', 'janjgir'),
            (t_cg, 'Dhamtari', 'dhamtari'),
            (t_cg, 'Mahasamund', 'mahasamund'),
            (t_cg, 'Kanker', 'kanker')
        ON CONFLICT (tenant_id, slug) DO NOTHING;

        -- Seed Story & Article for Chhattisgarh
        INSERT INTO stories (tenant_id, slug) VALUES (t_cg, 'raipur-visakhapatnam-corridor') RETURNING id INTO s_id;

        INSERT INTO articles (story_id, tenant_id, author_id, title, slug, excerpt, body, language, status, is_breaking, is_featured, published_at)
        VALUES (
            s_id, t_cg, author_id,
            'Raipur-Visakhapatnam Economic Corridor Accelerates Agro and Steel Exports',
            'raipur-visakhapatnam-economic-corridor-export-growth',
            'Direct high-speed expressway access to eastern seaports boosts industrial manufacturing and farmer value realizations across Chhattisgarh.',
            '[{"type":"paragraph","text":"The new economic expressway links central industrial zones with deepwater port terminals in record 7-hour transit times."}]'::jsonb,
            'en', 'published', FALSE, TRUE, NOW()
        ) ON CONFLICT (tenant_id, slug, language) DO NOTHING;
    END IF;

END $$;
