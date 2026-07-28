--
-- PostgreSQL database dump
--

\restrict atIFF51PQIho94DmdGMRhbOP31Fs8Va6nb4m6vvhUqWI80ZOnVbrMFhRe5Tex1O

-- Dumped from database version 16.14
-- Dumped by pg_dump version 16.14

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account_snapshots; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.account_snapshots (
    id bigint NOT NULL,
    account_id bigint NOT NULL,
    snap_date date NOT NULL,
    balance numeric(12,2) NOT NULL
);


ALTER TABLE public.account_snapshots OWNER TO themis;

--
-- Name: account_snapshots_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.account_snapshots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.account_snapshots_id_seq OWNER TO themis;

--
-- Name: account_snapshots_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.account_snapshots_id_seq OWNED BY public.account_snapshots.id;


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.accounts (
    id bigint NOT NULL,
    name text NOT NULL,
    kind text DEFAULT 'courant'::text NOT NULL,
    interest_rate numeric(5,2),
    deposit_limit numeric(12,2),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    owner_id bigint
);


ALTER TABLE public.accounts OWNER TO themis;

--
-- Name: accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.accounts_id_seq OWNER TO themis;

--
-- Name: accounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.accounts_id_seq OWNED BY public.accounts.id;


--
-- Name: analyses; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.analyses (
    id integer NOT NULL,
    month text NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    owner_id bigint
);


ALTER TABLE public.analyses OWNER TO themis;

--
-- Name: analyses_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.analyses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.analyses_id_seq OWNER TO themis;

--
-- Name: analyses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.analyses_id_seq OWNED BY public.analyses.id;


--
-- Name: budgets; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.budgets (
    id bigint NOT NULL,
    monthly_limit numeric(12,2) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    domain_id bigint,
    owner_id bigint,
    CONSTRAINT budgets_monthly_limit_check CHECK ((monthly_limit > (0)::numeric))
);


ALTER TABLE public.budgets OWNER TO themis;

--
-- Name: budgets_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.budgets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.budgets_id_seq OWNER TO themis;

--
-- Name: budgets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.budgets_id_seq OWNED BY public.budgets.id;


--
-- Name: credits; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.credits (
    id bigint NOT NULL,
    name text NOT NULL,
    monthly_amount numeric(12,2) NOT NULL,
    months_total integer NOT NULL,
    start_month character(7) NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    owner_id bigint,
    CONSTRAINT credits_monthly_amount_check CHECK ((monthly_amount > (0)::numeric)),
    CONSTRAINT credits_months_total_check CHECK ((months_total > 0))
);


ALTER TABLE public.credits OWNER TO themis;

--
-- Name: credits_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.credits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.credits_id_seq OWNER TO themis;

--
-- Name: credits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.credits_id_seq OWNED BY public.credits.id;


--
-- Name: domains; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.domains (
    id bigint NOT NULL,
    name text NOT NULL,
    parent_id bigint,
    color text DEFAULT '#66736C'::text NOT NULL,
    owner_id bigint,
    is_transfer boolean DEFAULT false NOT NULL
);


ALTER TABLE public.domains OWNER TO themis;

--
-- Name: domains_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.domains_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.domains_id_seq OWNER TO themis;

--
-- Name: domains_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.domains_id_seq OWNED BY public.domains.id;


--
-- Name: merchant_aliases; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.merchant_aliases (
    id bigint NOT NULL,
    owner_id bigint NOT NULL,
    match_key text NOT NULL,
    display text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    is_pattern boolean DEFAULT false NOT NULL
);


ALTER TABLE public.merchant_aliases OWNER TO themis;

--
-- Name: merchant_aliases_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.merchant_aliases_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.merchant_aliases_id_seq OWNER TO themis;

--
-- Name: merchant_aliases_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.merchant_aliases_id_seq OWNED BY public.merchant_aliases.id;


--
-- Name: rules; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.rules (
    id integer NOT NULL,
    pattern text NOT NULL,
    domain_id bigint NOT NULL,
    owner_id bigint,
    tags text DEFAULT ''::text NOT NULL
);


ALTER TABLE public.rules OWNER TO themis;

--
-- Name: rules_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.rules_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.rules_id_seq OWNER TO themis;

--
-- Name: rules_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.rules_id_seq OWNED BY public.rules.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.schema_migrations (
    name text NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO themis;

--
-- Name: sessions; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.sessions (
    token text NOT NULL,
    user_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL
);


ALTER TABLE public.sessions OWNER TO themis;

--
-- Name: settings; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.settings (
    key text NOT NULL,
    value jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    owner_id bigint
);


ALTER TABLE public.settings OWNER TO themis;

--
-- Name: statements; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.statements (
    id integer NOT NULL,
    filename text NOT NULL,
    period_start date,
    period_end date,
    opening_balance numeric(12,2),
    closing_balance numeric(12,2),
    parser text DEFAULT 'local'::text NOT NULL,
    validated boolean DEFAULT false NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    account_id bigint DEFAULT 1 NOT NULL,
    owner_id bigint
);


ALTER TABLE public.statements OWNER TO themis;

--
-- Name: statements_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.statements_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.statements_id_seq OWNER TO themis;

--
-- Name: statements_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.statements_id_seq OWNED BY public.statements.id;


--
-- Name: tag_folder_prefs; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.tag_folder_prefs (
    owner_id bigint NOT NULL,
    tag_name text NOT NULL,
    state text NOT NULL,
    CONSTRAINT tag_folder_prefs_state_check CHECK ((state = ANY (ARRAY['pinned'::text, 'hidden'::text])))
);


ALTER TABLE public.tag_folder_prefs OWNER TO themis;

--
-- Name: tags; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.tags (
    id bigint NOT NULL,
    name text NOT NULL,
    owner_id bigint
);


ALTER TABLE public.tags OWNER TO themis;

--
-- Name: tags_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.tags_id_seq OWNER TO themis;

--
-- Name: tags_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.tags_id_seq OWNED BY public.tags.id;


--
-- Name: taxes; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.taxes (
    id bigint NOT NULL,
    name text NOT NULL,
    amount numeric(12,2) NOT NULL,
    due_date date NOT NULL,
    paid boolean DEFAULT false NOT NULL,
    owner_id bigint,
    CONSTRAINT taxes_amount_check CHECK ((amount > (0)::numeric))
);


ALTER TABLE public.taxes OWNER TO themis;

--
-- Name: taxes_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.taxes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.taxes_id_seq OWNER TO themis;

--
-- Name: taxes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.taxes_id_seq OWNED BY public.taxes.id;


--
-- Name: transaction_tags; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.transaction_tags (
    transaction_id bigint NOT NULL,
    tag_id bigint NOT NULL
);


ALTER TABLE public.transaction_tags OWNER TO themis;

--
-- Name: transactions; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.transactions (
    id integer NOT NULL,
    statement_id integer,
    tx_date date NOT NULL,
    label text NOT NULL,
    amount numeric(12,2) NOT NULL,
    hash text NOT NULL,
    domain_id bigint,
    owner_id bigint,
    display_label text DEFAULT ''::text NOT NULL,
    merchant text DEFAULT ''::text NOT NULL,
    label_kind text DEFAULT ''::text NOT NULL
);


ALTER TABLE public.transactions OWNER TO themis;

--
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.transactions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.transactions_id_seq OWNER TO themis;

--
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.transactions_id_seq OWNED BY public.transactions.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: themis
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    username text NOT NULL,
    password_hash text NOT NULL,
    role text DEFAULT 'user'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT users_role_check CHECK ((role = ANY (ARRAY['admin'::text, 'user'::text])))
);


ALTER TABLE public.users OWNER TO themis;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: themis
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO themis;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: themis
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: account_snapshots id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots ALTER COLUMN id SET DEFAULT nextval('public.account_snapshots_id_seq'::regclass);


--
-- Name: accounts id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.accounts ALTER COLUMN id SET DEFAULT nextval('public.accounts_id_seq'::regclass);


--
-- Name: analyses id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.analyses ALTER COLUMN id SET DEFAULT nextval('public.analyses_id_seq'::regclass);


--
-- Name: budgets id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets ALTER COLUMN id SET DEFAULT nextval('public.budgets_id_seq'::regclass);


--
-- Name: credits id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.credits ALTER COLUMN id SET DEFAULT nextval('public.credits_id_seq'::regclass);


--
-- Name: domains id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains ALTER COLUMN id SET DEFAULT nextval('public.domains_id_seq'::regclass);


--
-- Name: merchant_aliases id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.merchant_aliases ALTER COLUMN id SET DEFAULT nextval('public.merchant_aliases_id_seq'::regclass);


--
-- Name: rules id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules ALTER COLUMN id SET DEFAULT nextval('public.rules_id_seq'::regclass);


--
-- Name: statements id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements ALTER COLUMN id SET DEFAULT nextval('public.statements_id_seq'::regclass);


--
-- Name: tags id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tags ALTER COLUMN id SET DEFAULT nextval('public.tags_id_seq'::regclass);


--
-- Name: taxes id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.taxes ALTER COLUMN id SET DEFAULT nextval('public.taxes_id_seq'::regclass);


--
-- Name: transactions id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions ALTER COLUMN id SET DEFAULT nextval('public.transactions_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Data for Name: account_snapshots; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.account_snapshots (id, account_id, snap_date, balance) FROM stdin;
2	2	2026-07-18	233.00
5	5	2026-07-18	1200.00
6	1	2026-07-20	58.00
7	7	2026-07-20	2962.00
8	8	2026-07-20	770.00
\.


--
-- Data for Name: accounts; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.accounts (id, name, kind, interest_rate, deposit_limit, created_at, owner_id) FROM stdin;
1	Compte courant	courant	\N	\N	2026-07-18 13:33:35.382182+00	1
2	Livret A	livret_a	\N	\N	2026-07-18 16:00:48.667578+00	1
5	Cryptomonaie	autre	\N	\N	2026-07-18 16:01:18.552838+00	1
6	Compte courant	courant	\N	\N	2026-07-18 17:42:32.763227+00	2
7	PEL	pel	\N	\N	2026-07-20 11:45:03.447029+00	1
8	PEA	bourse	\N	\N	2026-07-20 11:45:28.042904+00	1
\.


--
-- Data for Name: analyses; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.analyses (id, month, content, created_at, owner_id) FROM stdin;
1	2026-02	### Vue d'ensemble\n\nLe mois de février 2026 a été déficitaire avec un solde négatif de -121,02 €, ce qui indique une dépense légèrement supérieure aux revenus. La catégorie "Famille" représente plus de la moitié des dépenses totales.\n\n### Points d'attention\n\n- **Famille (3022,09 €)** : Cette dépense est très élevée par rapport au mois précédent (2170,00 €) et aux mois précédents. Il faut vérifier les détails pour comprendre la cause.\n- **Vie Professionnelle (3045,00 €)** : Ce poste a connu une forte augmentation depuis le mois de janvier 2026 (143,10 €). Il est important d'identifier l'origine de cette dépense.\n\n### Actions concrètes\n\n- **Réduire les frais "Famille"** : Si la dépense élevée en février est due à un événement unique comme une fête ou des cadeaux, envisagez de limiter ces dépenses futures. Réduction possible : 500 €.\n- **Analyser les frais professionnels** : Identifiez l'origine du montant élevé pour la "Vie Professionnelle". Si c'est un paiement unique, ajustez votre budget en conséquence. Réduction possible : 2901,90 € (pour revenir au niveau de janvier).\n- **Optimiser les transferts internes** : Le transfert interne de 5000 € est élevé et peut être réduit si non nécessaire. Réduction possible : 2500 €.\n\n### Tendance\n\nComparativement aux mois précédents, la dépense "Famille" a considérablement augmenté en février (3022,09 € contre 660 € en décembre et 2170 € en janvier). De même, les frais de "Vie Professionnelle" ont fortement grimpé à 3045 € depuis le niveau bas de janvier (143,10 €).	2026-07-18 15:21:39.506497+00	1
6	2026-05	### Vue d'ensemble\n\nLe mois de mai 2026 a été difficile financièrement, avec un solde négatif de -151.41 € malgré un revenu stable à 300.00 €. Les dépenses ont dépassé les revenus, principalement en raison d'une augmentation significative dans la catégorie Famille.\n\n### Points d'attention\n\n- **Famille (110.68 €)** : Cette dépense est anormalement élevée par rapport aux mois précédents et nécessite une vérification pour s’assurer qu’il n’y a pas de coûts imprévus ou inutiles.\n- **Alimentation (155.40 €)** : Bien que stable, cette catégorie reste importante et pourrait être réduite si d'autres dépenses sont élevées.\n\n### Actions concrètes\n\n1. **Réduire les frais de Famille** : Identifiez les dépenses inutiles ou excessives dans la catégorie Famille (par exemple, abonnements non utilisés) pour économiser 50 €.\n2. **Planifier des repas à domicile** : Réduisez vos dépenses en alimentation de 155.40 € à 130.00 € par mois en cuisinant plus souvent et en évitant les achats impulsifs.\n3. **Réévaluer le budget Logement** : Si possible, négociez votre loyer ou vos factures d'énergie pour réduire la dépense de logement à 85.00 €.\n\n### Tendance\n\nComparativement aux mois précédents, les dépenses en Famille ont considérablement augmenté (110.68 € contre 390.95 € en avril et 1950.00 € en mars), ce qui est inhabituel et doit être investigué. Les dépenses d'Alimentation sont stables mais restent élevées par rapport aux revenus, tandis que les dépenses de Logement ont fluctué fortement (de 2947.34 € en avril à 97.34 € en mars et février).	2026-07-20 11:47:32.30573+00	1
7	2026-04	### Vue d'ensemble\n\nLe mois d'avril 2026 a été marqué par un solde négatif de -2463.70 €, principalement en raison d'un excès important des dépenses sur les revenus. La catégorie Logement représente une part disproportionnée du budget total.\n\n### Points d'attention\n\n- **Logement (2947.34 €)** : Ce poste est anormalement élevé par rapport aux mois précédents, où il oscillait autour de 1000 €.\n- **Famille (390.95 €)** : Une forte augmentation depuis le dernier mois qui était à 1950 € mais avec une variation importante d'un mois sur l'autre.\n- **Transport (6.12 €)** : Un montant très bas comparé aux dépenses habituelles, ce qui peut indiquer un manque de frais engagés.\n\n### Actions concrètes\n\n1. **Réduire les coûts liés au logement** : Identifier et négocier les factures élevées (par exemple, la taxe d'habitation ou le loyer) pour ramener ce poste à un niveau plus raisonnable de 1500 €.\n2. **Équilibrer les dépenses familiales** : Établir une moyenne des frais familiaux et limiter les dépenses excessives à environ 800 € par mois.\n3. **Augmenter les frais de transport** : Planifier au moins un trajet supplémentaire par mois pour atteindre un montant plus réaliste d'environ 50 €.\n\n### Tendance\n\nComparativement aux trois derniers mois, le poste Logement est significativement plus élevé en avril (2947.34 € contre une moyenne de 1680 €), tandis que les dépenses dans la catégorie Famille oscillent fortement entre 1950 € et 390 €. Les autres catégories restent stables ou légèrement inférieures à leur niveau habituel, sauf pour le Transport qui est très bas (6.12 € contre une moyenne de 48 €).	2026-07-20 11:51:37.851324+00	1
2	2026-06	### Vue d'ensemble\n\nCe mois-ci, la situation financière est saine avec un solde positif de 1882.42 €. Cependant, les dépenses dans certaines catégories ont connu des variations importantes.\n\n### Points d'attention\n\n- **Famille** : Une hausse significative à 2780.10 € (EN HAUSSE de 240%) mérite une attention particulière.\n- **Divertissement** : Les dépenses dans cette catégorie ont bondi à 978.98 € (EN HAUSSE de 445%), ce qui est inhabituel.\n- **Logement** : Une baisse importante à 97.34 € (EN BAISSE de 91%) indique un changement notable dans les frais liés au logement.\n\n### Actions concrètes\n\n1. **Réévaluer les dépenses familiales** : Identifiez et réduisez les coûts non essentiels dans la catégorie Famille (2780 €). Par exemple, si des frais médicaux ou éducationnels sont en cause, vérifiez s'ils peuvent être négociés ou reportés.\n   \n2. **Contrôler le divertissement** : Réduisez les dépenses de divertissement à un niveau plus réaliste (978 €). Évitez les achats impulsifs et planifiez des activités gratuites ou peu coûteuses.\n\n3. **Investiguer la baisse du logement** : Vérifiez pourquoi le coût du logement a baissé de manière significative (97 €) et assurez-vous que cela ne résulte pas d'une erreur ou d'un report temporaire des frais.\n\n### Tendance\n\nComparativement aux mois précédents, les dépenses dans la catégorie Famille ont considérablement augmenté (+240%), tandis que celles de Divertissement ont également connu une hausse importante (+445%). Les coûts liés au Logement sont en baisse (-91%) et ceux relatifs à l'Alimentation ont diminué (-70%). La catégorie Vie Professionnelle reste stable.	2026-07-21 07:27:03.070794+00	1
\.


--
-- Data for Name: budgets; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.budgets (id, monthly_limit, created_at, domain_id, owner_id) FROM stdin;
2	50.00	2026-07-18 15:39:32.526109+00	23	1
3	150.00	2026-07-18 15:59:04.66369+00	24	1
5	200.00	2026-07-18 16:00:29.534534+00	26	1
6	50.00	2026-07-18 17:34:06.66885+00	30	1
7	200.00	2026-07-18 18:57:22.219284+00	27	1
\.


--
-- Data for Name: credits; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.credits (id, name, monthly_amount, months_total, start_month, note, owner_id) FROM stdin;
2	Cofidis soeur	63.00	4	2026-05		1
3	Paypal Billet Avion	287.00	4	2026-06		1
\.


--
-- Data for Name: domains; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.domains (id, name, parent_id, color, owner_id, is_transfer) FROM stdin;
14	Transport	\N	#3F8F7B	\N	f
15	Alimentation	\N	#C77D2E	\N	f
16	Santé et Bien-être	\N	#D6455D	\N	f
17	Divertissement	\N	#8A5CD1	\N	f
18	Épargne et Investissements	\N	#2E8B57	\N	f
19	Vie Professionnelle	\N	#3D6B99	\N	f
21	Autres	\N	#8A8A99	\N	f
1	Logement	\N	#4C6FBF	\N	f
5	Famille	\N	#B5548A	\N	f
22	Logement	\N	#4C6FBF	1	f
23	Transport	\N	#3F8F7B	1	f
24	Alimentation	\N	#C77D2E	1	f
25	Santé et Bien-être	\N	#D6455D	1	f
26	Divertissement	\N	#8A5CD1	1	f
27	Épargne et Investissements	\N	#2E8B57	1	f
28	Vie Professionnelle	\N	#3D6B99	1	f
29	Famille	\N	#B5548A	1	f
30	Autres	\N	#8A8A99	1	f
32	Télécommunications	\N	#8C6D2F	1	f
33	Amis	\N	#A63A28	1	f
34	Logement	\N	#4C6FBF	2	f
35	Transport	\N	#3F8F7B	2	f
36	Alimentation	\N	#C77D2E	2	f
37	Santé et Bien-être	\N	#D6455D	2	f
38	Divertissement	\N	#8A5CD1	2	f
39	Épargne et Investissements	\N	#2E8B57	2	f
40	Vie Professionnelle	\N	#3D6B99	2	f
41	Famille	\N	#B5548A	2	f
42	Autres	\N	#8A8A99	2	f
43	Transfert interne	\N	#7A8290	2	t
31	Transfert interne	\N	#c51135	1	t
\.


--
-- Data for Name: merchant_aliases; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.merchant_aliases (id, owner_id, match_key, display, created_at, is_pattern) FROM stdin;
1	1	Vers Ben-mohamed Madi T	Vers Trade Republic	2026-07-20 10:48:43.796667+00	f
2	1	Les Uli	Carrefour les Ulis	2026-07-20 10:49:00.07437+00	f
3	1	Lessonsharing Mtb- - -30f:1- - -1	Lessonsharing	2026-07-20 10:49:08.629226+00	f
4	1	Offre Essentiel	Frais Bancaire	2026-07-20 10:49:19.689856+00	f
5	1	Africa Exotique Les	Africa Exotique Les Ulis	2026-07-20 10:49:32.621604+00	f
6	1	Nyx Justeatemps Rois	Machine à café	2026-07-20 10:49:45.578101+00	f
7	1	Lessonsharing Mtb- - -22f:1- - -1	Lessonsharing	2026-07-20 10:49:51.168336+00	f
8	1	Vers Madi Toumani Sakin	Vers Sakina	2026-07-20 10:49:57.514441+00	f
9	1	Ulis Viandes Les Uli	Boucherie Les Ulis	2026-07-20 10:50:20.333257+00	f
10	1	Nyx Stcommerce Paris	Machine à café Paris	2026-07-20 10:50:34.102104+00	f
11	1	Vers Mme Jing An	Loyer	2026-07-20 10:51:04.173608+00	f
12	1	Vers M.madi Tarik	Vers Tarik	2026-07-20 10:51:11.468523+00	f
13	1	De Madi Toumani Sakina	De Sakina	2026-07-20 10:51:21.222186+00	f
14	1	De Mile Aya N’goran	De Sonia	2026-07-20 10:51:29.645563+00	f
15	1	De M. Madi Tarik	De Tarik	2026-07-20 10:51:39.749016+00	f
16	1	Snbru Air08223037157	Frais Brussels Airline	2026-07-20 10:52:02.071354+00	f
17	1	Snbru Air Roissy	Frais Brussels Airline	2026-07-20 10:52:12.882846+00	f
18	1	Vers Aya N’goran	Vers Sonia	2026-07-20 10:52:25.965093+00	f
19	1	Shein Dublin	Shein	2026-07-20 10:52:36.111427+00	f
20	1	De M. Madi Toumani Ben-moh	Vers Compte courant	2026-07-20 10:53:16.423305+00	f
21	1	Web M. Madi Toumani Ben-moha	Vers Livret A	2026-07-20 10:53:27.385787+00	f
22	1	Idfm Idfm - - Mand Mandate-reference00000000000055687 7fr42277457385 Core	IDF Mobilité	2026-07-20 11:02:29.894474+00	f
23	1	Acheel France Ref-927b2080281923c168gfgwbh4 Core	Assurance habitation Acheel France	2026-07-20 11:02:54.917849+00	f
24	1	Acheel France Ref-1cbbb9880770f24695gfgwbh4 Fr457z787f911 Core	Assurance habitation Acheel France	2026-07-20 11:03:09.649848+00	f
25	1	Acheel France Ref-2f3e4b269ac60316d9gfgwbh4 Core	Assurance habitation Acheel France	2026-07-20 11:03:12.489135+00	f
26	1	Acheel France Ref-996928ef59fa47a80bgfgwbh4 Fr4527787f911 Core	Assurance habitation Acheel France	2026-07-20 11:03:15.293355+00	f
27	1	Acheel France Ref-81821f2daf9edes013gfgwbh4 Core	Assurance habitation Acheel France	2026-07-20 11:03:18.190951+00	f
28	1	Tiktok Shop Seller D	Tiktok Shop	2026-07-20 11:03:38.946919+00	f
29	1	Acheel France Core	Assurance habitation Acheel France	2026-07-20 11:03:52.09711+00	f
30	1	Idfm Idfm - - Mand Ate-reference000000000000556877 Core	IDF Mobilité	2026-07-20 11:04:23.506352+00	f
31	1	Lessonsharing - - -mtb 00005f:1- - -1	Lessonsharing	2026-07-20 11:04:34.429489+00	f
32	1	Lessonsharing Mtb- - -30f:1- - -2	Lessonsharing	2026-07-20 11:04:35.225929+00	f
33	1	Lessonsharing - - -mtb 00002f:1- - -1	Lessonsharing	2026-07-20 11:04:35.985027+00	f
34	1	Lessonsharing - - -mtbf:1- - -1	Lessonsharing	2026-07-20 11:04:37.060676+00	f
35	1	Viandes Du Argent	Boucherie Argenteuil	2026-07-20 11:04:49.729333+00	f
36	1	Ornikar Paris	Ornikar Paris	2026-07-20 11:04:56.515321+00	f
37	1	Bouygues Telecom	Bouygues Telecom Mobile	2026-07-20 11:05:31.839914+00	f
38	1	Free	Free	2026-07-20 11:05:32.456856+00	f
39	1	Deliveroo	Deliveroo	2026-07-20 11:05:50.325233+00	f
40	1	Engie	Engie	2026-07-20 11:05:50.776179+00	f
41	1	Uber	Uber	2026-07-20 11:05:51.064414+00	f
42	1	Carrefour	Carrefour	2026-07-20 11:05:51.368797+00	f
43	1	Burger King	Burger King	2026-07-20 11:05:51.732512+00	f
44	1	Service Navigo Paris	Service Navigo Paris	2026-07-20 11:05:52.212893+00	f
45	1	PayPal	Paypal	2026-07-20 11:05:53.622144+00	f
46	1	Amazon	Amazon	2026-07-20 11:05:54.171227+00	f
47	1	SNCF	SNCF	2026-07-20 11:05:58.785647+00	f
48	1	Vers Assjad Saida	Vers Assjad	2026-07-20 11:06:06.794725+00	f
49	1	Royal Hotel Monrovia	Royal Hotel Monrovia	2026-07-20 11:06:19.465662+00	f
50	1	Vers Ambassade De Liber	Ambassade Du Liberia	2026-07-20 11:06:37.646946+00	f
51	1	Action	Action	2026-07-20 11:06:38.230361+00	f
52	1	Sc.pedra Alta Paris	Pedra Alta Paris	2026-07-20 11:06:51.027698+00	f
53	1	Ret Les Ulis 14h30	Retrait Distributeur Les Ulis	2026-07-20 11:07:07.38877+00	f
54	1	Ret Les Ulis 09h28	Retrait Distributeur Les Ulis	2026-07-20 11:07:10.751744+00	f
55	1	Heetch Transport Ba	Heetch	2026-07-20 11:07:55.70874+00	f
56	1	Five Pizza Les Ui	Five Pizza Les Uis	2026-07-20 11:07:56.203234+00	f
57	1	Heetch” Transport Ba	Heetch	2026-07-20 11:07:56.382661+00	f
58	1	Caresquick Topaze	Caresquick Topaze	2026-07-20 11:07:56.600568+00	f
59	1	La Toscana Paris	La Toscana Paris	2026-07-20 11:07:57.04389+00	f
60	1	Mcdo Lohearches Vinc	Mcdo	2026-07-20 11:07:57.807822+00	f
61	1	Yamanashi Paris	Yamanashi Paris	2026-07-20 11:07:58.882516+00	f
62	1	Sloann Paris	Sloann Paris	2026-07-20 11:08:00.382106+00	f
63	1	Lessonsharing Mtb- - -31f: - -1	Lessonsharing	2026-07-20 11:08:26.175813+00	f
64	1	Lessonsharing - - -mtb 00004f:1- - -1	Lessonsharing	2026-07-20 11:08:26.418386+00	f
65	1	Lessonsharing Mtb- - -31f:1- - -1	Lessonsharing	2026-07-20 11:08:26.610581+00	f
66	1	Lessonsharing Mtb- - -27f:1- - -1	Lessonsharing	2026-07-20 11:08:26.775853+00	f
67	1	Lessonsharing - - -mtb 00006f:1- - -1	Lessonsharing	2026-07-20 11:08:26.942678+00	f
68	1	Lessonsharing Mtb- - - F:1- - -1	Lessonsharing	2026-07-20 11:08:27.1177+00	f
69	1	Lessonsharing - - -mtb 00003f:1- - -1	Lessonsharing	2026-07-20 11:08:27.630238+00	f
70	1	Lessonsharing - - -1	LessonSharing	2026-07-21 07:24:52.34603+00	f
71	1	Lessonsharing - - - -1	LessonSharing	2026-07-21 07:24:54.075523+00	f
72	1	Snbru	SNBRU	2026-07-21 07:24:58.781582+00	f
73	1	Hippo Bercy Paris	Hippo Bercy	2026-07-21 07:25:01.665384+00	f
74	1	Wero Vers Aleksandar S	Wero Aleksandar	2026-07-21 07:25:03.718376+00	f
75	1	African Fashion F Pa	African Fashion	2026-07-21 07:25:04.58617+00	f
76	1	Idfm Idfm - - Mand Mandate- Core	IDFM Core	2026-07-21 07:25:06.704141+00	f
77	1	Wero Vers Vadijiguiba Di	Wero Vadijiguiba	2026-07-21 07:25:10.000432+00	f
78	1	Pizza Andiamo Les Ui	Andiamo Pizza	2026-07-21 07:25:12.480504+00	f
79	1	Food Les Ulis	Les Ulis Food	2026-07-21 07:25:14.060848+00	f
80	1	Idfm Idfm - - Mand Ate- Core	IDFM Core	2026-07-21 07:25:15.322665+00	f
81	1	Ret Les Ulis	Retrait Distributeur Les Ulis	2026-07-21 07:25:36.11573+00	f
82	1	Lessonsharing - - -2	Lessonsharing	2026-07-21 07:26:06.652186+00	f
83	1	Phcie Courdimanc Les	Phcie Courdimanc	2026-07-21 07:26:07.419984+00	f
\.


--
-- Data for Name: rules; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.rules (id, pattern, domain_id, owner_id, tags) FROM stdin;
57	Amazon Payments	26	1	amazon
7	ENGIE	1	\N	
8	EDF	1	\N	
9	LOYER	1	\N	
59	M. Madi Tarik	29	1	frere
23	SALAIRE	19	\N	
14	UBER EATS	17	\N	
15	DELIVEROO	17	\N	
16	MCDONALD	17	\N	
17	ORANGE	17	\N	
18	FREE MOBILE	17	\N	
19	AMAZON PRIME	17	\N	
20	SPOTIFY	17	\N	
21	NETFLIX	17	\N	
1	MONOPRIX	21	\N	
2	INTERMARCHE	21	\N	
3	LIDL	21	\N	
4	AUCHAN	21	\N	
5	LECLERC	21	\N	
6	CARREFOUR	21	\N	
10	re:\\yESSO\\y	21	\N	
11	TOTALENERGIES	21	\N	
12	UBER	21	\N	
13	SNCF	21	\N	
22	PHARMACIE	21	\N	
24	VIREMENT	21	\N	
25	FRAIS	21	\N	
26	COTISATION	21	\N	
27	CARREFOUR	24	1	
29	AUCHAN	24	1	
30	LIDL	24	1	
32	MONOPRIX	24	1	
33	FRANPRIX	24	1	
34	CASINO	24	1	
35	LOYER	22	1	
36	EDF	22	1	
37	ENGIE	22	1	
38	VEOLIA	22	1	
39	SNCF	23	1	
44	NETFLIX	26	1	
47	PHARMACIE	25	1	
48	DOCTEUR	25	1	
49	FRAIS BANCAIRES	30	1	
50	COTISATION	30	1	
53	Vir Inst vers Madi Toumani Sakin	29	1	soeur
54	Prlv Engie Mandat Core	22	1	facture,abonnement,energie,engie
65	Mile Aya N’goran	33	1	
70	LESSONSHARING	28	1	salaire,lessonsharing
72	Hotel Monrovia	29	1	voyage,mariage,jessica
73	Vir Inst de Madi Toumani Sakina	29	1	soeur
71	TARIK	29	1	frere
61	Nyx*stcommerce Paris	28	1	cafe,ipssi
74	PAYPAL	26	1	paypal
76	COURDIMANC	25	1	
77	ASSJAD	29	1	niece
78	GORAN	33	1	
31	Carte X4519 Action Les Ulis	24	1	courses,action
42	RATP	23	1	ratp
55	Nyx*justeatemps Rois	28	1	cafe,ipssi
64	X4519 Les Uli	24	1	courses,carrefour
58	URSSAF	28	1	urssaf,taxes,cotisations
43	Web M. Madi Toumani Ben-moha	31	1	deplacement,livret-a
28	Carte X4519 Ulis Viandes Les Uli	24	1	courses,boucherie
80	Prlv Idfm IDFM	23	1	navigo
52	Vir Inst vers Ben-mohamed Madi T	29	1	trade-republic
62	Prlv Offre Essentiel	30	1	abonnement,frais-bancaires,credit agricole
63	Prlv Free Telecom Free Hautdebit	22	1	facture,abonnement,internet,free
51	De M. Madi Toumani Ben-moh	31	1	deplacement,compte-courant
45	Prlv Paypal (europe) $.a r.l. et Cie	26	1	paypal,credit
46	Carte X4519 Amazon	26	1	amazon
66	Mme Jing An	22	1	loyer
68	DELIVEROO	24	1	fast-food,deliveroo
41	Virement De M. Madi Toumani Ben-moh	31	1	deplacement,compte-courant
60	Acheel France	22	1	facture,abonnement,assurance,acheel
56	Prlv Bouygues Telecom	32	1	facture,abonnement,mobile,bouygues
69	Mens.pel	27	1	pel
79	Burger King	24	1	fast-food,burger-king
67	Service Navigo	23	1	navigo
40	UBER	23	1	uber
75	Africa Exotique	24	1	courses,africain
82	CARREFOUR	36	2	
83	LECLERC	36	2	
84	AUCHAN	36	2	
85	LIDL	36	2	
86	INTERMARCHE	36	2	
87	MONOPRIX	36	2	
88	FRANPRIX	36	2	
89	CASINO	36	2	
90	LOYER	34	2	
91	EDF	34	2	
92	ENGIE	34	2	
93	VEOLIA	34	2	
94	SNCF	35	2	
95	UBER	35	2	
96	TOTALENERGIES	35	2	
97	RATP	35	2	
98	ESSENCE	35	2	
99	NETFLIX	38	2	
100	SPOTIFY	38	2	
101	DISNEY	38	2	
102	PHARMACIE	37	2	
103	DOCTEUR	37	2	
104	FRAIS BANCAIRES	42	2	
105	COTISATION	42	2	
106	MONROVIA	29	1	
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.schema_migrations (name, applied_at) FROM stdin;
001_init.sql	2026-07-18 13:33:35.35836+00
002_budgets.sql	2026-07-18 13:33:35.376151+00
003_settings.sql	2026-07-18 13:33:35.379833+00
004_features.sql	2026-07-18 13:33:35.382182+00
005_domains_seed.sql	2026-07-18 13:33:35.404818+00
006_domain_taxonomy.sql	2026-07-18 13:33:35.405641+00
007_domains_dedup.sql	2026-07-18 13:33:35.410821+00
008_categories.sql	2026-07-18 13:33:35.413077+00
009_rules_unique.sql	2026-07-18 13:33:35.414855+00
010_data_isolation.sql	2026-07-18 13:33:35.416429+00
011_budget_owner_unique.sql	2026-07-18 13:33:35.432647+00
012_rule_tags.sql	2026-07-18 13:33:35.434214+00
013_transfer_category.sql	2026-07-18 13:33:35.434893+00
014_display_labels.sql	2026-07-20 09:55:19.603055+00
015_label_kind.sql	2026-07-20 09:55:19.606142+00
016_merchant_aliases.sql	2026-07-20 09:55:19.607042+00
017_tag_pin.sql	2026-07-20 09:55:19.610917+00
018_alias_pattern.sql	2026-07-21 07:21:03.753969+00
\.


--
-- Data for Name: sessions; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.sessions (token, user_id, expires_at) FROM stdin;
gphM6xILyr4oy33ACUYqK74LbiHwvTWYquu7ATNcWps	1	2026-08-17 17:46:39.065744+00
\.


--
-- Data for Name: settings; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.settings (key, value, updated_at, owner_id) FROM stdin;
ai_config	{"provider": "ollama", "ollama_url": "http://10.238.149.218:11434", "ollama_model": "qwen2.5:14b", "anthropic_model": "claude-sonnet-4-6", "auto_categorize": true}	2026-07-21 07:23:44.613869+00	\N
dashboard	[{"id": "w1", "size": "full", "type": "kpi", "config": {}}, {"id": "w2", "size": "full", "type": "flux", "config": {"months": 12}}, {"id": "wa", "size": "half", "type": "attention", "config": {}}, {"id": "w4", "size": "half", "type": "solde", "config": {"months": 12}}, {"id": "w100_2989", "size": "half", "type": "trend", "config": {"months": 12, "category": "Autres"}}, {"id": "w3", "size": "half", "type": "categories", "config": {}}, {"id": "w5", "size": "half", "type": "top", "config": {"count": 5}}, {"id": "w6", "size": "half", "type": "subs", "config": {}}]	2026-07-21 19:38:25.963239+00	1
\.


--
-- Data for Name: statements; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.statements (id, filename, period_start, period_end, opening_balance, closing_balance, parser, validated, uploaded_at, account_id, owner_id) FROM stdin;
1	Releve n001 du 21-01-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2025-12-22	2026-01-21	68.18	611.81	local	t	2026-07-18 13:34:23.07909+00	1	1
2	Releve n002 du 20-02-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-01-21	2026-02-20	611.81	112.46	local	f	2026-07-18 14:42:58.237521+00	1	1
3	Releve n003 du 20-03-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-02-20	2026-03-20	112.46	208.71	local	t	2026-07-18 15:41:31.90789+00	1	1
4	Releve n004 du 21-04-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-03-20	2026-04-21	208.71	53.02	local	t	2026-07-18 15:44:49.806446+00	1	1
5	Releve n005 du 21-05-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-04-21	2026-05-21	53.02	306.46	local	t	2026-07-18 15:45:25.873851+00	1	1
7	Releve n006 du 22-06-2026-CCHQ 43609805111 M MADI TOUMANI.pdf	2026-05-21	2026-06-22	306.46	38.47	local	t	2026-07-18 18:17:23.945522+00	1	1
\.


--
-- Data for Name: tag_folder_prefs; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.tag_folder_prefs (owner_id, tag_name, state) FROM stdin;
1	taxes	hidden
1	deliveroo	hidden
1	courses	hidden
\.


--
-- Data for Name: tags; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.tags (id, name, owner_id) FROM stdin;
184	permis	1
119	move	1
82	hotel	1
81	transfert	1
86	pizza	1
199	frigo	1
227	restaurant	1
373	alimentation	1
45	ipssi	1
67	taxi	1
1	deplacement	1
19	compte-courant	1
204	credit	1
187	paypal	1
245	uber	1
1846	pere	1
1794	viandes	1
33	urssaf	1
35	cotisations	1
59	frais-bancaires	1
179	credit agricole	1
14	facture	1
15	abonnement	1
39	assurance	1
40	acheel	1
65	boucherie	1
16	energie	1
17	engie	1
36	frere	1
331	burger-king	1
68	loyer	1
64	courses	1
262	carrefour	1
78	soins	1
235	cadeau	1
87	salaire	1
1283	africain	1
88	lessonsharing	1
1796	cofidis	1
1860	medicaments	1
13	soeur	1
62	internet	1
63	free	1
70	navigo	1
234	tiktok	1
172	mariage	1
30	mobile	1
31	bouygues	1
71	fast-food	1
72	deliveroo	1
1863	administratif	1
66	remboursement	1
122	livret-a	1
369	pret	1
355	action	1
75	pel	1
24	cafe	1
638	starlink	1
237	visa	1
240	shein	1
183	niece	1
329	coiffeur	1
1835	erreur	1
173	jessica	1
1948	voyage-liberia	1
171	voyage	1
637	projet	1
1899	demenagement	1
32	amazon	1
1966	horace	1
34	taxes	1
1969	caf	1
6	trade-republic	1
1964	pea	1
\.


--
-- Data for Name: taxes; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.taxes (id, name, amount, due_date, paid, owner_id) FROM stdin;
1	Cotisations Juin	1510.00	2026-07-31	f	1
2	Cotisation Juillet	395.00	2026-08-31	f	1
3	Impots	4040.00	2026-09-30	f	1
\.


--
-- Data for Name: transaction_tags; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.transaction_tags (transaction_id, tag_id) FROM stdin;
46	1
16	1
45	6
40	6
44	13
43	14
43	15
43	16
43	17
42	1
42	19
39	1
39	19
38	1
38	19
37	24
31	24
34	24
35	24
36	14
36	15
36	30
36	31
33	32
32	33
32	34
32	35
30	36
28	14
28	15
28	39
28	40
27	24
17	24
24	24
37	45
31	45
34	45
35	45
27	45
17	45
24	45
26	15
26	59
23	14
23	15
23	62
23	63
20	67
18	68
14	70
13	71
13	72
12	71
12	72
11	75
50	67
51	67
83	82
84	82
101	86
1	87
1	88
2	87
2	88
3	87
3	88
4	87
4	88
5	87
5	88
78	87
78	88
69	87
69	88
70	87
70	88
71	87
71	88
72	87
72	88
73	87
73	88
74	87
74	88
75	87
75	88
76	87
76	88
77	87
77	88
82	122
82	1
16	19
54	1
54	19
97	1
97	19
98	1
98	19
108	1
108	19
46	19
58	1
58	19
66	1
66	19
110	1
110	19
111	1
111	19
68	75
83	171
83	172
83	173
84	171
84	172
84	173
26	179
88	15
88	59
88	179
90	183
90	184
118	71
164	187
167	81
200	187
206	67
207	187
212	86
228	13
228	199
221	13
81	36
164	204
207	204
56	187
56	204
167	187
213	187
217	187
208	187
6	227
211	227
119	227
177	227
115	227
55	187
55	66
121	245
120	245
106	33
106	34
106	35
105	33
105	34
105	35
104	14
104	15
104	16
104	17
102	70
52	64
52	262
100	64
100	262
140	64
140	262
171	64
171	262
195	64
195	262
224	64
224	262
226	64
226	262
219	64
219	65
218	64
204	71
204	331
196	64
196	65
182	64
182	355
103	64
103	65
47	64
47	355
22	64
22	262
166	13
166	369
187	13
187	66
80	13
80	373
124	87
124	88
173	87
173	88
174	87
174	88
191	33
191	34
191	35
150	16
150	14
150	15
150	17
201	16
201	14
201	15
201	17
227	16
227	14
227	15
227	17
92	14
92	15
92	30
92	31
141	14
141	15
141	30
141	31
198	14
198	15
198	30
198	31
225	14
225	15
225	30
225	31
63	6
67	6
87	6
94	6
144	6
181	6
96	32
163	32
176	32
184	32
210	32
210	66
203	32
203	66
183	32
116	1
116	19
132	1
132	19
136	1
136	19
142	1
142	19
143	1
143	19
148	1
148	19
151	1
151	19
154	1
154	19
155	1
155	19
159	1
159	19
165	1
165	19
168	1
168	19
170	1
170	19
179	1
179	19
192	1
192	19
199	1
199	19
202	1
202	19
209	1
209	19
133	15
133	59
133	179
190	15
190	59
190	179
223	15
223	59
223	179
126	1
126	122
158	1
158	122
85	14
85	15
85	62
85	63
127	14
127	15
127	62
127	63
189	14
189	15
189	62
189	63
222	14
222	15
222	62
222	63
180	68
86	14
86	15
86	39
86	40
130	14
130	15
130	39
130	40
188	14
188	15
188	39
188	40
220	14
220	15
220	39
220	40
123	75
175	75
215	75
161	70
7	245
8	245
206	70
129	71
129	72
134	71
134	72
152	71
152	72
160	71
160	72
185	71
185	72
49	64
49	1283
139	64
139	1283
194	64
194	1283
99	64
99	1283
48	227
10	369
10	66
285	14
285	15
285	62
285	63
289	1
289	122
299	1
299	19
308	1
308	19
310	14
310	15
310	16
310	17
306	14
306	15
306	30
306	31
297	15
297	59
297	179
272	70
279	87
279	88
280	87
280	88
281	87
281	88
282	87
282	88
283	87
283	88
284	87
284	88
277	36
303	64
303	262
278	14
278	15
278	39
278	40
271	71
271	331
291	71
291	331
276	75
307	183
293	33
293	34
293	35
311	33
311	34
311	35
287	36
312	32
312	1796
312	13
305	234
305	235
305	173
304	187
304	66
301	187
301	204
301	171
301	173
300	32
300	173
197	204
197	187
197	171
197	173
205	329
300	637
300	638
298	329
296	237
296	171
296	173
292	240
292	235
292	173
290	235
205	262
205	64
214	204
214	187
214	1835
295	204
295	187
295	171
295	173
295	1835
62	245
62	1835
146	1846
147	1846
294	1794
294	65
275	64
275	262
274	64
274	262
128	64
128	262
59	78
59	173
59	235
53	171
53	1860
157	173
157	172
157	1863
21	66
21	369
60	1835
60	171
60	173
61	1835
61	171
61	173
9	6
9	173
29	6
29	173
41	6
41	173
64	6
64	173
57	6
57	173
79	6
79	173
89	6
89	173
91	6
91	173
95	6
95	173
109	6
109	173
109	637
109	1899
112	6
112	173
112	637
112	1899
117	6
117	173
122	6
122	173
125	6
125	173
131	6
131	173
135	6
135	173
137	6
137	173
138	6
138	173
149	6
149	173
162	6
162	173
169	6
169	173
172	6
172	173
178	6
178	173
186	6
186	173
193	6
193	173
216	6
216	173
286	6
286	173
302	6
302	173
309	6
309	173
309	1948
309	171
288	6
288	173
288	1948
288	171
15	32
15	6
19	6
19	32
65	6
65	1964
93	6
93	1966
107	6
107	34
107	1969
113	6
113	1964
114	6
114	1964
156	6
156	1964
153	6
153	1964
273	6
273	1964
\.


--
-- Data for Name: transactions; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.transactions (id, statement_id, tx_date, label, amount, hash, domain_id, owner_id, display_label, merchant, label_kind) FROM stdin;
37	1	2026-01-15	Carte X4519 Nyx*justeatemps Rois 14/01	-0.50	748e8db9fe07066a7907b8bd97e02d95cce7623f9fa714842bcc7b6185d4c26c	28	1	Machine à café	Nyx Justeatemps Rois	
31	1	2026-01-12	Carte X4519 Nyx*justeatemps Rois 12/01	-0.50	f55e2cfea00e105f48eb0394c223248c65247990d0fcaf22313f297df4cacc83	28	1	Machine à café	Nyx Justeatemps Rois	
34	1	2026-01-13	Carte X4519 Nyx*justeatemps Rois 13/01	-0.50	f7c1815c919710fccfd8741568ef72151d3c9fd0c9f5e9a0836c16de6e1b4ec2	28	1	Machine à café	Nyx Justeatemps Rois	
44	1	2026-01-19	Vir Inst vers Madi Toumani Sakin	-20.00	2d8ee317eeeeeda4709172f5012cfe53552766d58ee9c30300c458d99b11c45d	29	1	Vers Sakina	Vers Madi Toumani Sakin	
27	1	2026-01-09	X4519 Nyx*stcommerce Paris 08/01	-0.70	0827f75dbe73f60642fc5e37ac0a9dfdbdc43a683b930f7001de367d863c88da	28	1	Machine à café Paris	Nyx Stcommerce Paris	
17	1	2026-01-05	Carte X4519 Nyx*stcommerce Paris 05/01	-0.70	7f1713f944d57c5545f1448ddc77137fc36ad1635921266f722c9e251eb3af9f	28	1	Machine à café Paris	Nyx Stcommerce Paris	
24	1	2026-01-08	X4519 Nyx*stcommerce Paris 08/01	-0.70	8a8d7d8b8dac0b0dc67bc4bcb9b2300ccad4d7827e62f62ce8233e20ac98a197	28	1	Machine à café Paris	Nyx Stcommerce Paris	
21	1	2026-01-06	Vir Inst de Mile Aya N’goran	900.00	2b6f85f955b364cc1e2b36f9a8bc4f2c8f7ee133cd93e250ccfe74d8f9e08185	33	1	De Sonia	De Mile Aya N’goran	
10	1	2026-01-01	Virement Vir Inst de Mile Aya N’goran	300.00	7720b564d9de37cbcc2a7d3ca48625bbcba91d21f51e04130cb568719038c346	33	1	De Sonia	De Mile Aya N’goran	
30	1	2026-01-10	Vir Inst de M. Madi Tarik	50.00	fc7576b95382909d912f421f66826cdfd96980265b42772e0153a50b13a45f94	29	1	De Tarik	De M. Madi Tarik	
46	1	2026-01-21	De M. Madi Toumani Ben-moh	400.00	1c194cbab45ee6c151145ca666ab1565f46b248da6692d5cb1aba2ac14bee4ff	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
38	1	2026-01-17	Virement De M. Madi Toumani Ben-moh	600.00	a5d0738b18a5e27b031508e85eb70119b02372a94ea0deefdd8d4a7c152243e3	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
39	1	2026-01-17	De M. Madi Toumani Ben-moh	100.00	ebd5d66d63a3746c71cf2b79a3f3236fe0622892f7f119d3df48be51ddfb7917	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
36	1	2026-01-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	f0837dc01a41b58cee9af27c9e38e7a6209491b79542cad02688945a52c28274	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
13	1	2026-01-02	Carte X4519 Deliveroo 75009 Pari 01/01	-27.04	3b7786a7dd6440375361441ae423f64c17e24d5ec6fe62cf0e7bedbe562465ad	24	1	Deliveroo	Deliveroo	
12	1	2026-01-02	Carte X4519 Deliveroo 75009 Pari 01/01	-27.44	371b8f2c52504d0c78c761b9f44dd2f00f5e23a1b28e157eccc90e9d1e8435bc	24	1	Deliveroo	Deliveroo	
43	1	2026-01-19	Prlv Engie Mandat Core	-61.80	afc28de00503614adb62be9eaa4260654ec93bd40d0b43000dac87117f58cff4	22	1	Engie	Engie	
8	1	2025-12-29	Carte X4519 Ubr* Pending.uber.co 27/12	-20.91	3a45063f750b733a8a9349c8803219cadacef3a183b093562b5c9aaa5b52083e	23	1	Uber	Uber	
14	1	2026-01-02	Carte X4519 Service Navigo Paris 01/01	-22.70	0be9297c2e582afb7b08a304a6c6f6e5049509965679ef0f32d52a502a635488	23	1	Service Navigo Paris	Service Navigo Paris	
33	1	2026-01-13	Carte X4519 Amazon Payments Payl 10/01	-99.89	6f6b2835b457479dd264124446361ee48b3524bff972f281f0a3e1e63497ac56	26	1	Amazon	Amazon	
32	1	2026-01-12	Carte X4519 URSSAF Marseille 09/01	-139.00	2fe82768eebdf4e42a0f79b753c3149ee4409e6a6baefeb34732d7c748f209ee	28	1	Cotisation URSSAF	URSSAF	tax
11	1	2026-01-01	Virement Mens.pel 44690429796 01/26	-50.00	5f7975a01a7f7a9bd47a08450e62b8c26ce22cd54723b349b77d4d92cdd4e9fa	27	1	Versement PEL	PEL	savings
22	1	2026-01-06	X4519 Les Uli 05/01	-46.92	a9a1c7dd1876661284e1b3c41208d4ee3783fa58e9e0cfa0c88c47fa1131b76f	24	1	Carrefour les Ulis	Les Uli	
20	1	2026-01-05	Vir Inst Wero vers Vadijiguiba Di	-10.00	0b700f76c19e6b1ad7f0a3d0b65b9639ab173ba137baae8147fada76804c1a53	23	1	Wero Vadijiguiba	Wero Vers Vadijiguiba Di	
100	2	2026-02-17	Carte X4519 Les Uli 16/02	-36.04	0b5b61fd0e052f5f66edeb98331bbcfc62ea68db5c40b800f3a74b0450272588	24	1	Carrefour les Ulis	Les Uli	
88	2	2026-02-09	Prlv Offre Essentiel	-6.00	825a19f5bf17b33ce6041c726a4b0b35ca11e420c3a111808f5c9d8b037a09ca	30	1	Frais Bancaire	Offre Essentiel	
103	2	2026-02-17	Carte X4519 Ulis Viandes Les Uli 16/02	-37.38	103ac2eddd2cebea608f81032068404af04369678b899f5e30ac386e78cf9705	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
61	2	2026-01-26	X4519 Snbru Air Roissy 23/01	-150.00	7fec3c701955d369d334079f4077318142b64b2ace541229175ece9cb5ee0138	23	1	Frais Brussels Airline	Snbru Air Roissy	
58	2	2026-01-25	De M. Madi Toumani Ben-moh	330.00	f4c3393bc83a63ca55f9d427599d1282379da9073637f677006b9259e298a124	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
4	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-02-10-1	310.00	cb52507f4a2ccafcf53f28c98e085af9f1a5428fb140773b3004489ded9f6707	28	1	LessonSharing	Lessonsharing - - -1	
3	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-03-10-1	620.00	0f66c9bd92497ef6ca47dde725db95f2031bb9ee77ef1e344bc61d69a36d4bba	28	1	LessonSharing	Lessonsharing - - -1	
48	2	2026-01-22	Carte X4519 African Fashion F Pa 21/01	-47.40	1a649321ba2c305ddb24bc90efe25118b7b6fa112dd7900de65d4f6c71aa53de	33	1	African Fashion	African Fashion F Pa	
52	2	2026-01-22	X4519 Les Uli 21/01	-28.13	ab06f0e6306074ad31953ca2bd1435f4a7bec9bf2806d367067e2b5cd7510162	24	1	Carrefour les Ulis	Les Uli	
26	1	2026-01-09	Prlv Offre Essentiel	-6.00	5b7ef440df283bef8e9f1256e0650c467c6e24b360aea1e6abb4f82f54500823	30	1	Frais Bancaire	Offre Essentiel	
49	2	2026-01-22	Carte X4519 Africa Exotique Les 21/01	-11.84	a1f78f7b464f38f58792b276b53253e543730a5df0b1cc19ab0e48720a3a715b	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
55	2	2026-01-23	Virement Paypal Europe $S.a-r.l. et Cie S. YYW1047781999815YYW1047781999815/PAYPAL	162.50	f3292029bd4930e65c795807c586f7ef3d0c26dadcc2c25d991532317cba0224	26	1	Paypal	PayPal	
84	2	2026-02-04	Carte X4519 Royal Hotel Monrovia 26/01	-21.10	65e17f8c6e90fdd97d1a32fec2abe92d537e62e7f58be8390f19002176e71e6f	29	1	Royal Hotel Monrovia	Royal Hotel Monrovia	
51	2	2026-01-22	Carte X4519 Heetch”* Transport Ba 21/01	-16.00	4b45959177ac136976a171663bdee64c3fe25d295e1ca81356a477e6024518ba	23	1	Heetch	Heetch” Transport Ba	
59	2	2026-01-26	Carte X4519 Caresquick Topaze 24/01	-15.90	691f8b2c1f7db96869892766ab140d21efc48be27f83974ce2d22b86d1ebc8d2	25	1	Caresquick Topaze	Caresquick Topaze	
68	2	2026-02-01	Virement Mens.pel 44690429796 02/26	-50.00	d244ab61968f87a48f10913e1e73315dd31b4c0d75b8f28bb00ccad2a11b762a	27	1	Versement PEL	PEL	savings
105	2	2026-02-18	Carte X4519 URSSAF Marseille 17/02	-2291.00	ee552ebcb319614f73929a7aeafc93e3b39219088d3cd70729d37c8494fd2870	28	1	Cotisation URSSAF	URSSAF	tax
140	3	2026-03-13	X4519 Les Uli 12/03	-51.40	99a98960ba1a900e5104cb0f792306684bb5e434ee88087b251455c74b046922	24	1	Carrefour les Ulis	Les Uli	
86	2	2026-02-06	Prlv Acheel France REF-1CBBB9880770F24695GFGWBH4 FR457Z787F911 Core	-12.47	961ea78d18210e8de49469e821468b9595cf48e8a528928178a3261baf433470	22	1	Assurance habitation Acheel France	Acheel France Core	
104	2	2026-02-17	Prlv Engie Mandat Core	-54.88	3a080721448516d6ae2ede8f29a4375ba5e135fdf3d44f2f88550af230bc6c69	22	1	Engie	Engie	
133	3	2026-03-09	Prlv Offre Essentiel	-6.00	a5010453db428530a2b8a8cf6832923d496b0ae6930687af12a21f6541f37b83	30	1	Frais Bancaire	Offre Essentiel	
139	3	2026-03-13	X4519 Africa Exotique Les 12/03	-29.50	ca5ec60b93f64bbc37538314e3963ec597e41ea0ef297bdef2a2ed3ff4e9b617	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
116	3	2026-02-25	De M. Madi Toumani Ben-moh	200.00	baf171382c3abeaf99b3f5c4209cc53bd2191c3989ca5e965a8999eb6548fb28	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
132	3	2026-03-08	De M. Madi Toumani Ben-moh	180.00	a6673ae73463e756d9cbeaa7db980560196fb34751c0c55174910a0bead83fac	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
136	3	2026-03-10	De M. Madi Toumani Ben-moh	200.00	cbe6ec5be5b613d9b4b2e651b6ca6274ea7eaaf56ec909fe495cff306d50a718	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
142	3	2026-03-14	De M. Madi Toumani Ben-moh	50.00	ce01063b1aa242e5b5634a8a7b30aed9b7f2374afd509bc4b8097b8d6be20889	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
143	3	2026-03-14	Virement De M. Madi Toumani Ben-moh	100.00	d8646b3acb21a81bc0979155ba04b0403c7d430ce3a17e105c71c2d02bff9c34	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
126	3	2026-03-04	Web M. Madi Toumani Ben-moha	-1000.00	1f66c389b93a6bce081a6cc1051d55f8b214557d27b4f04d2fe601e4919f927a	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
90	2	2026-02-10	Carte X4519 Ornikar 75019 Paris 09/02	-7.99	3e218a7d7daf422088ca1c2f530a29f61813dc0398ab0a2ec2c092d188342b2f	29	1	Ornikar Paris	Ornikar Paris	
141	3	2026-03-13	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	2850fa332943a70afc91e547a9ecb92664fdd8867b5eb9e1006d896bd81acd07	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
127	3	2026-03-05	Prlv Free Telecom Free Hautdebit 145690719847939382 FR83277459654 Core	-29.99	53d3100e5159c67f414113ed79aa2cc98469dac5bc190a93ea1529238fd3ea45	22	1	Free	Free	
129	3	2026-03-06	Carte X4519 Deliveroo 75009 Pari 05/03	-22.39	85fff82b578be5c1fc12d9e49d84f703fecac5ab39184f96854f5a875fe7c7be	24	1	Deliveroo	Deliveroo	
70	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-08-11-1	1050.00	b51e4c4ddc7e13fe3a08351d158966c9a06a9e54e5256eff2b31d08149a34113	28	1	LessonSharing	Lessonsharing - - -1	
60	2	2026-01-26	Carte X4519 Snbru AIR08223037157 23/01	-187.65	79f46248f53fe8c21e355bb96f7422a80f1b60d9f57c50db54a492758ee7c9d3	23	1	SNBRU	Snbru	
77	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-22-11-2	590.00	5091b843f3cca5cc5f11363889b35bb22a0a1a9cab6163d044031b3b82871c44	28	1	Lessonsharing	Lessonsharing - - -2	
53	2	2026-01-22	Carte X4519 Phcie Courdimanc Les 21/01	-19.80	18098506adea77bc947fd60843cea1ea5347b41e9637e7789ba2e38012c8d7ec	25	1	Phcie Courdimanc	Phcie Courdimanc Les	
15	1	2026-01-02	Virement Vir Inst vers Ben-mohamed Madi T	-210.00	26fac8ebc18444bb2843cf37e1c6158c8d6f07b57c0f93812e754c176bbc64c1	26	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
108	2	2026-02-19	Virement De M. Madi Toumani Ben-moh	750.00	594c20e97ff9c1649fc2bc26110df780f414cd967bc6b90b52019bcad4052cf4	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
85	2	2026-02-05	Prlv Free Telecom Free Hautdebit 144933115747939382 FR83277459654 Core	-29.99	4436b61f7817dcf807ce60e931fbcb0ab19db8804a7719c886874175741c5432	22	1	Free	Free	
110	2	2026-02-20	De M. Madi Toumani Ben-moh	980.00	4da68e0171bc00a8d5960b7041df4f6cc0f6669ad6ba958df5e475396330e5e8	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
92	2	2026-02-12	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	9f0e99c8dbf5a0529630685323a5a740438ec0839ca06747be1e5b0732207532	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
62	2	2026-01-26	X4519 Ubr* Pending.uber.co 23/01	-52.99	5b981d13b69d722f7f039e71d60b7433d763c2c57b82e2eca26a407eaa586eb3	23	1	Uber	Uber	
102	2	2026-02-17	Carte X4519 Service Navigo Paris 16/02	-2.55	861c2db48f41609a8cc1f44a5cbb2f0e9b426a2879979619d215127335f8e4ab	23	1	Service Navigo Paris	Service Navigo Paris	
96	2	2026-02-16	Carte X4519 Amazon Payments Payl 09/02	-36.18	52c3786c980bc1189b0325ebad36e205bf5fd15095aa62a23f41835b483109fb	26	1	Amazon	Amazon	
147	3	2026-03-16	Carte X4519 Sncf-voyageurs Paris 14/03	-39.00	85117a0fc06d7af1be2a930b9049a72e029b267e88674de24feb6ae38893d0b8	23	1	SNCF	SNCF	
119	3	2026-02-26	Carte X4519 Sloann Paris 12 25/02	-19.50	addc7f016af52755730b095c25d6db1760387a2f5867ab9ebe5cb608eeb36003	28	1	Sloann Paris	Sloann Paris	
123	3	2026-03-01	Virement Mens.pel 44690429796 03/26	-50.00	a1d862e5b9380954fd2bc0b6e12ebdbc18ab4aa4838ebe499720450c47155fd8	27	1	Versement PEL	PEL	savings
120	3	2026-02-26	Carte X4519 Ubr”* Pending.uber.co 25/02	-7.20	8c39e68f2757eb10f8deaff2b59d6e2deb933378868905152d8ea590b2083bbb	23	1	Uber	Uber	
171	4	2026-03-31	X4519 Les Uli 30/03	-34.53	f8ae39c3506ad392facbd569d9785e56fb0c099e6c737e0b29b5fd8ad7650d22	24	1	Carrefour les Ulis	Les Uli	
121	3	2026-02-27	Carte X4519 Ubr”* Pending.uber.co 25/02	-28.91	60f718be85b3aa0801d359a34a625858f4b9dce1608d3f117a749d5f7e77844d	23	1	Uber	Uber	
195	4	2026-04-13	X4519 Les Uli 10/04	-61.72	e5d935cd879929b4ae1341b4b970362aded9d198eb6073190d16ed07fbb465f6	24	1	Carrefour les Ulis	Les Uli	
190	4	2026-04-09	Prlv Offre Essentiel	-6.00	9be3661107ecfd1df35ac4fe69feb26ed3cf9d560bb6d14266080e6ba77d67c3	30	1	Frais Bancaire	Offre Essentiel	
194	4	2026-04-13	X4519 Africa Exotique Les 10/04	-10.55	1b77baaf2e5e4461150b243a00dc1b04fecd22083984c8f28019c95477524ba9	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
166	4	2026-03-26	Vir Inst vers Madi Toumani Sakin	-650.00	1464b57414435a94a03563ae446ea3241f62d859bc8e4e2b29f192c9c9af43b7	29	1	Vers Sakina	Vers Madi Toumani Sakin	
196	4	2026-04-13	Carte X4519 Ulis Viandes Les Uli 10/04	-39.27	c9d4a9fd281784096a2737f0c620d346b92fd771ef20219bb2ea92683af6f1aa	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
180	4	2026-04-03	Virement Vir Inst vers Mme Jing An	-2850.00	ef09e4d4989cd0945928d5e44bfa98372b5c41715e22fcdfc42f2512f1b03f99	22	1	Loyer	Vers Mme Jing An	
187	4	2026-04-08	Virement Vir Inst de Madi Toumani Sakina	650.00	ca8399e295c0b62d9f4e480f590f16e129a10ac4f177ce073401f0f5dff73ffa	29	1	De Sakina	De Madi Toumani Sakina	
151	3	2026-03-19	De M. Madi Toumani Ben-moh	250.00	cbfae6942e2c7455570ccb905055c69f3d912247367938cbbb7fcaefd6644078	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
155	4	2026-03-22	De M. Madi Toumani Ben-moh	60.00	36c672fe934edc2c1f0a89d195357821736c28f4a0a670c77b3ac81905db723e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
165	4	2026-03-26	De M. Madi Toumani Ben-moh	650.00	29170ff6fe6167691e57855fcf2440b190591d2030541bdceeaa84ba7517d659	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
168	4	2026-03-28	De M. Madi Toumani Ben-moh	200.00	5b7e382b0c9bd4e792c71b169a46b540aef75812faf9a3474b84da8188dcaddc	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
170	4	2026-03-30	De M. Madi Toumani Ben-moh	100.00	0edec6a73df9c23c467f04bc00989d30dab691809423de94cdc96fc11789554a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
192	4	2026-04-11	De M. Madi Toumani Ben-moh	50.00	02f6177edfbc886eb86432f9253c677be4332ca1d10f256f0e72d2d22f1e0a56	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
199	4	2026-04-15	De M. Madi Toumani Ben-moh	250.00	149c037a5366e7863e56fc629b27d39e53db67fb026efb95fff827728a8c619e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
158	4	2026-03-22	Web M. Madi Toumani Ben-moha	-30.00	72a099afa3cff15782a4059fb011f7e47f5b65167d1647e883094ef6ae6acb21	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
188	4	2026-04-08	Prlv Acheel France Core	-12.47	f486bf3def29010439235c3a5852185cbb9956891e05670f9d0d299c2e131a36	22	1	Assurance habitation Acheel France	Acheel France Core	
189	4	2026-04-08	Prlv Free Telecom Free Hautdebit 146452883347939382 FR83277459654 Core	-29.99	195ed1953779b9c8059c289dcba69a2d336b5af470a0d29cf8532b1fdb002d60	22	1	Free	Free	
160	4	2026-03-23	Carte X4519 Deliveroo 75009 Pari 22/03	-25.24	0c44643271857b8eaaf3a935f717deafc637bb9b2629ec8ef568ec1444bfe5cc	24	1	Deliveroo	Deliveroo	
161	4	2026-03-23	X4519 Service Navigo Paris 22/03	-2.55	8fa0a73a45f3a39b10cad7d715d713b7019effa309ac721bce9e71bb8cddcb4c	23	1	Service Navigo Paris	Service Navigo Paris	
167	4	2026-03-27	Virement Vir Inst de Paypal	162.50	34bae17528cb373f46d5626a9438953b5fdf6881cec07fbeb9869f69e4a8cb8c	31	1	Paypal	PayPal	
176	4	2026-04-02	Carte X4519 Amazon Payments Payl 31/03	-39.11	938299285acc9037c3c1cd8438a6be37118f0f0f8d2975bc1b544a167f26c29c	26	1	Amazon	Amazon	
184	4	2026-04-07	Carte X4519 Amazon Payments Payl 02/04	-38.98	b40154a1320b13c835e64dde2531d8befd5680349fe6b349b16ff1a006862aa9	26	1	Amazon	Amazon	
183	4	2026-04-07	Carte X4519 Amazon Eu SARL Payli 02/04	-18.59	0754c2dd8f3a0c0d38624aaedec5ff49275adc3fb668a135ee58097a33e3a79b	26	1	Amazon	Amazon	
182	4	2026-04-07	Carte X4519 Action Les Ulis 03/04	-9.00	78db49e20bce828f9e09f217e0f95669ecf99fd3e84b6cec4ddcb28b692a0383	24	1	Action	Action	
177	4	2026-04-02	Carte X4519 La Toscana Paris 01/04	-15.00	8b6e15785e236302e1c7663cb7dda348891082d32c575cd788247b8dfcb0f61e	28	1	La Toscana Paris	La Toscana Paris	
118	3	2026-02-26	Carte X4519 Mcdo Lohearches Vinc 26/02	-12.15	3fda4331500425f687b18050690c4329e7234719d7de6594cd4606f0d758ea0b	24	1	Mcdo	Mcdo Lohearches Vinc	
175	4	2026-04-01	Mens.pel 44690429796 04/26	-50.00	1ef6a98f48fcad7e531ff59c6ccf219870db77ab9368e3a83e23b8f6efe59c7e	27	1	Versement PEL	PEL	savings
40	1	2026-01-17	Virement Vir Inst vers Ben-mohamed Madi T	-80.00	8288e9a089a257b56a8401597bd55895dc263637a82bd23515acafdfea2531b3	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
174	4	2026-04-01	Virement Lessonsharing MTB-2026-03-31F:02-02-1	175.00	baac190d8fb9411472ae2778588fcbda61e079bf1205d590ce2e0b855b18baf8	28	1	LessonSharing	Lessonsharing - - -1	
101	2	2026-02-17	X4519 Pizza Andiamo Les UI 16/02	-9.00	fc86af56947c4ef23ed97e32ad200e87cd45dfb60808802400400b6bcc0339c7	24	1	Andiamo Pizza	Pizza Andiamo Les Ui	
19	1	2026-01-05	Vir Inst vers Ben-mohamed Madi T	-80.00	daa4ed45d20d5a4f5946c06563c03c280bec6044dfc50ad05e8b92f361d01e43	24	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
128	3	2026-03-06	X4519 Carrefour.fr Mondevi 05/03	-63.18	cfb639b497c3e7e6bc7e613ae02e21efc451996844d125412549d7dbd9ef7a9f	24	1	Carrefour	Carrefour	
146	3	2026-03-16	X4519 Lw*sncf Connect Pari 13/03	-41.10	95d06c24f6745a664266e2888bc882af62c721ad763931348248ee85d9366c1f	23	1	SNCF	SNCF	
277	7	2026-06-03	Virement Vir Inst de M. Madi Tarik	300.00	12c51872f348e194c4da397b1fcd41d6d3e018b1f6dc23a2784fdf9096551c9f	29	1	De Tarik	De M. Madi Tarik	
202	4	2026-04-18	De M. Madi Toumani Ben-moh	40.00	506fcc98154e682545e79bb47e13a832ef2826880cb7e58302a75cdb19e6318a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
209	5	2026-04-25	De M. Madi Toumani Ben-moh	150.00	5b37ec7de08a6b8f1215df1ae3804ce9bec0b53bfcf82bd8c2c52b32dc7b16c4	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
225	5	2026-05-13	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-11.99	a648ef643c35c99f950273b9ef1105b9eb99085ca6f965fbd7ef78d7e5732d0c	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
222	5	2026-05-06	Prlv Free Telecom Free Hautdebit 147098249247939382 Core	-29.99	96fa685944142c298f92d1c47724dd582f40f0310e351959727f8162eebb5cc7	22	1	Free	Free	
227	5	2026-05-19	Prlv Engie Mandat Core	-54.88	ab72a42c561000ebb0d822c9dd9f1adc45fa05ef439e9c8c0cefc1dde291c4b3	22	1	Engie	Engie	
274	7	2026-05-26	Carte X4519 Mp*carrefour Les Uli 23/05	-21.31	862bcdb2820dc1869dffff6925ffa6ad9a9c51f5b396329b8ba7dda7b9c11a6c	24	1	Carrefour	Carrefour	
275	7	2026-05-29	Carte X4519 Mp*carrefour Les Uli 28/05	-19.00	36320f4a458e5cb09c466baf3708b5a84225e26e74e538107fa4d9d7d43835a6	24	1	Carrefour	Carrefour	
271	7	2026-05-22	Carte X4519 Burger King Les Ulis 21/05	-23.10	00fb7390e45003c329f9c24277160d23419b182955aad6ab613362e4552126a5	24	1	Burger King	Burger King	
200	4	2026-04-16	Prlv Paypal Europe $S.a-r.l. et Cie S. 1049622956887/PAYPAL567J2253W2NHA LU967ZZZ0000000000000000058 Core	-50.00	a2af1f539d6f13836e309a1ef7fb5b75b08e824bd512e3a669f9908c60ed31bb	30	1	Paypal	PayPal	
207	4	2026-04-21	Prlv Paypal (europe) $.a r.l. et Cie, 1049729158879 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-162.49	159d8fe2589ae086751de28e0fb6f04a7ff4d136d8e763f522631ac3349c1f2e	30	1	Paypal	PayPal	
210	5	2026-04-27	Avoir Carte X4519 Amazon Payment 24/04	24.99	f49143e8484104de6766470b741f4f69323564f3600fc039da919d102b24d2c8	30	1	Amazon	Amazon	
203	4	2026-04-20	Avoir Carte X4519 Amazon Payment 18/04	27.98	97568e704d03a1d4e15007cc67de513f4a520d250120d65d9583eae6e5bf482e	30	1	Amazon	Amazon	
211	5	2026-04-27	Carte X4519 Sc.pedra Alta Paris 25/04	-45.00	b96c3a1eb5016bcd959f3534d928ccc64b35265d9803264d825f3ec699f29730	33	1	Pedra Alta Paris	Sc.pedra Alta Paris	
212	5	2026-04-28	Carte X4519 Five Pizza Les UI 27/04	-17.10	7f4e2312ab3a87c9d61b26578f9414d003161147900ab941ff325e878dc95f17	26	1	Five Pizza Les Uis	Five Pizza Les Ui	
215	5	2026-05-01	Virement Mens.pel 44690429796 05/26	-50.00	e5563379d0c7c8128be15a3b05ac8ce0dd39e1268332502548faf18b8802d37f	27	1	Versement PEL	PEL	savings
157	4	2026-03-22	Wero vers Aleksandar S	-140.00	ae9597f0d4f66fcdf4740ec0f628dc275af84cd758aeaabbe57ca0e979dee566	29	1	Wero Aleksandar	Wero Vers Aleksandar S	
218	5	2026-05-04	Carte X4519 H.t. Food Les Ulis 30/04	-8.80	d829463c6f73c499b601dfeeb19904fa87d13c70bc767e514dfaff249d302e9a	24	1	Les Ulis Food	Food Les Ulis	
276	7	2026-06-01	Virement Mens.pel 44690429796 06/26	-50.00	3d68970445c11ef896dc560be7264be696ae059d59aa2b30fffe99c01251cf59	27	1	Versement PEL	PEL	savings
303	7	2026-06-11	X4519 Les Uli 10/06	-39.46	f183b924f495a4bcb61f1290efe94b45d484350c20f97f6209594b94f8d82205	24	1	Carrefour les Ulis	Les Uli	
297	7	2026-06-09	Prlv Offre Essentiel	-6.00	1713b5f2cc2c34ddf3128af25d9d0a957b4a7a7232af99e5bfa5ef1244f8bb1f	30	1	Frais Bancaire	Offre Essentiel	
35	1	2026-01-14	Carte X4519 Nyx*justeatemps Rois 14/01	-0.50	ab519dc46e22d2f89b0fd85827ee648e062371229c10f381daee8e8082b87929	28	1	Machine à café	Nyx Justeatemps Rois	
18	1	2026-01-05	Virement Vir Inst vers Mme Jing An	-2850.00	b45c1aa3cc1cef11a2a8cbc831d0a96b679f2b4f77b7bd52772a1fb99b57d89e	22	1	Loyer	Vers Mme Jing An	
292	7	2026-06-08	Carte X4519 Shein Dublin 07/06	-39.43	5522969788a2ebb991c199040bab614650d245144b7da02f565090deaffcb08c	29	1	Shein	Shein Dublin	
16	1	2026-01-04	Virement De M. Madi Toumani Ben-moh	800.00	0e1767051c00591eee3b6c0d024607d981ebbc4bfae4065fa9d1826fb7df5573	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
42	1	2026-01-19	Virement De M. Madi Toumani Ben-moh	200.00	7e55d7d9dfb3dd5519af040e3351af479e713e93d031008025274e21c0ada58f	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
54	2	2026-01-23	Virement De M. Madi Toumani Ben-moh	200.00	9701cc97b070cf50262972fb7332c5d49b15f8fc5a814e91ac747e55782f868c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
299	7	2026-06-10	De M. Madi Toumani Ben-moh	200.00	f9c833af81dbcf6a5d7d729d805cef36e6a29acbc4beec902de0d2263db6a9e0	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
5	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-04-10-1	310.00	fc0c5cde9927c3ef3fdf3243c3a1ba9df208bbfe01083caa3c74ac7fe025834a	28	1	LessonSharing	Lessonsharing - - -1	
279	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTBF:1-19-03-1	560.00	ae141762f5c8508fd1f5f11aad8c4d642bf28cc8b4e920f45adac9e3edca6307	28	1	LessonSharing	Lessonsharing - - - -1	
6	1	2025-12-29	Carte X4519 Hippo Bercy Paris 27/12	-170.80	39acd6b4d304231c5bc92aa9a018c7579a88c3bc3a0df9d9fc6cec8f3547fa92	33	1	Hippo Bercy	Hippo Bercy Paris	
205	4	2026-04-20	Ret DAB X4519 Les Ulis 18/04 14H30	-20.00	03eedebe422481fc80848603f707126bca5baa6d1f5e7d8cf6e7e1adc460482a	30	1	Retrait Distributeur Les Ulis	Ret Les Ulis	
226	5	2026-05-18	X4519 Les Uli 15/05	-39.43	8bc39cd10aab4d283f6de1ac87ee5072db9627643418ac5e95ba220fe3794c1d	24	1	Carrefour les Ulis	Les Uli	
223	5	2026-05-09	Prlv Offre Essentiel	-6.00	c616d15df65cdb799f20d4d1caed18357eb7dd1bf43e42b64a7a9986f255af55	30	1	Frais Bancaire	Offre Essentiel	
219	5	2026-05-04	Carte X4519 Ulis Viandes Les Uli 02/05	-35.73	1f42410c3c30c5b42bcdb6bdb56181d788a3abb811c33a1865a18afc08e905ad	24	1	Boucherie Les Ulis	Ulis Viandes Les Uli	
221	5	2026-05-06	Vir Inst de Madi Toumani Sakina	300.00	83e15c79135019cd3fcc705c3a3f5d3a72b6fbe8c19b9a7dd19fec3f835fea34	29	1	De Sakina	De Madi Toumani Sakina	
285	7	2026-06-05	Prlv Free Telecom Free Hautdebit 147851605447939382 Core	-29.99	d2a231177c78374caf88094d211174d4f1756b94826d75d1ce3145e9a113c1bc	22	1	Free	Free	
310	7	2026-06-17	Prlv Engie Mandat Core	-54.88	216ee2d4ee4a4d667c3064761a647b0c680345f868ad2eff31565f153567e017	22	1	Engie	Engie	
7	1	2025-12-29	Carte X4519 Uber “trip Amsterdam 28/12	-5.19	04000d099619dad07b35ac6ea1c0d6faaede2ea8d0d406e8bc4f23a153bdaaff	23	1	Uber	Uber	
291	7	2026-06-08	Carte X4519 Burger King Les Ulis 05/06	-12.00	0e6115b62df271058f121355a4557ff637e97c00a678999597738eb73619be9b	24	1	Burger King	Burger King	
295	7	2026-06-08	Prlv Paypal (europe) $.a r.l. et Cie, 1050779745906 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-391.89	1f0b39e26d4bc939f8a273d0a9435f26a7da419e1f1815a90c99544baeea5c0d	26	1	Paypal	PayPal	
304	7	2026-06-12	Vir Inst de Paypal	197.79	9d3967e0dd229d9cf46629c23f485c27321e09a00c5cbc649c7972362876c3cc	30	1	Paypal	PayPal	
312	7	2026-06-22	Carte X4519 Cofidis Amazon Ville 20/06	-63.68	f5fcfdbb5cf97ecf5af1830673a6c973d6159348ad10c0c33868658d1f22b69c	29	1	Amazon	Amazon	
300	7	2026-06-10	Carte X4519 Amazon Payments Pari 08/06	-186.61	21729ff974875318fbec55a21b33df5784f33550a2f7c1c2c797bfada050c38c	29	1	Amazon	Amazon	
307	7	2026-06-14	Vir Inst vers Assjad Saida	-105.00	49065f520c12d399470cfd97dea31427c9d48662c6b8deef1b0a8ad4e91bc3eb	29	1	Vers Assjad	Vers Assjad Saida	
296	7	2026-06-08	Virement Vir Inst vers Ambassade de Liber	-300.00	5be9872b869c10c00e6d8ee0802d5b944ecc6cc3d49be86342af902fa2c92970	26	1	Ambassade Du Liberia	Vers Ambassade De Liber	
47	2	2026-01-22	Carte X4519 Action Les Ulis 21/01	-11.83	2187d32bc2308795449d84a938a571080c4a4e91b5ce512e5f00735afc27c1b6	24	1	Action	Action	
293	7	2026-06-08	Carte X4519 URSSAF Idf Montreuil 05/06	-358.00	5689924b29524d95ab74ba17d342ffe75bc5ad6c8d28c6240f84f0fe648538c6	28	1	Cotisation URSSAF	URSSAF	tax
311	7	2026-06-19	Carte X4519 URSSAF Idf Montreuil 18/06	-17.00	bcf514e2889f4e19917dc63e2de8b0d8be5b71abb2800b79dce11aa2d931ef56	28	1	Cotisation URSSAF	URSSAF	tax
106	2	2026-02-18	Carte URSSAF Marseille 17/02	-754.00	27e67c554cd7e8308435ea288bf19e65222926a68a283618996188309837ce5c	28	1	Cotisation URSSAF	URSSAF	tax
191	4	2026-04-09	Carte X4519 URSSAF Idf St Denis 08/04	-361.00	5cfa8f17786f0c0f27e584c9b828c5631fc9f65c3cca096d2107dad2354d576e	28	1	Cotisation URSSAF	URSSAF	tax
99	2	2026-02-17	Carte X4519 Africa Exotique Les 16/02	-10.96	3f3aa668f031ca5396049865bbcadf0fde458cd109e48d36dd9258f4861650f9	24	1	Africa Exotique Les Ulis	Africa Exotique Les	
80	2	2026-02-03	Vir Inst vers Madi Toumani Sakin	-100.00	aca3adde8ef70a5231b8a0f4e290b1c5f92cff3dd12770b9c6d03a0d12e93a0a	29	1	Vers Sakina	Vers Madi Toumani Sakin	
81	2	2026-02-03	Vir Inst vers M.madi Tarik	-100.00	5d007a65d275e443a5348b3e7d331e56dffa2468635fe2a11ee808fe39d35cbd	29	1	Vers Tarik	Vers M.madi Tarik	
66	2	2026-01-29	De M. Madi Toumani Ben-moh	500.00	765b07c472fe44a14a1e20e8c16f115a7e0aaa57811c8f632c3b8fb77c2e48ad	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
98	2	2026-02-17	Virement De M. Madi Toumani Ben-moh	400.00	867e075cbc1f47df32069dc47283e25252077562699be828cbea1588dbc8519a	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
148	3	2026-03-17	Virement De M. Madi Toumani Ben-moh	50.00	14e4583fc220688f3e9ad52924e7ca2467166c4c8d4272bf4ecedb337aa1405e	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
154	4	2026-03-22	De M. Madi Toumani Ben-moh	150.00	b650ab801e653e41ee7b1c33ea11f4c937e33a9d104d00082089ed39f2c3fa20	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
159	4	2026-03-23	De M. Madi Toumani Ben-moh	100.00	359ef2148957714350ae16bf62fe62f2b1c3c0caed69320ccc5f33dee45f47ae	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
82	2	2026-02-03	Web M. Madi Toumani Ben-moha	-5000.00	b94c5b597eb9e9a4e7a464552388406cd1d773bbc903b57aed58480d75a8ecb6	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
130	3	2026-03-06	Prlv Acheel France REF-2F3E4B269AC60316D9GFGWBH4 Core	-12.47	e65ddc0780565b31e5518333df2e223287ee76a0f81318e6c95fff09d475adde	22	1	Assurance habitation Acheel France	Acheel France Core	
152	3	2026-03-20	Carte X4519 Deliveroo 75009 Pari 20/03	-26.29	9144908c70d85e6842ac195c8c870a42d0edb8a000dbd5ab16a6ba05dbf097e9	24	1	Deliveroo	Deliveroo	
71	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-22-12-1	1520.00	e5aa4b7a68b8bdce8c64c63f6381d977ed3e0da02d1939b488a0af0ee8ce8a88	28	1	LessonSharing	Lessonsharing - - -1	
284	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00004F:1-07-04-1	1210.00	f6294c2114c8fe65343a699c4f22578c22d7c7feee99b83e06472293c67193ca	28	1	LessonSharing	Lessonsharing - - - -1	
1	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22 F:1-27-09-1	280.00	e84851419d8533a560c0ec447688c90ac20cb2a90a65f0eff33883c9d3f6059b	28	1	LessonSharing	Lessonsharing - - - -1	
298	7	2026-06-09	Ret DAB X4519 Les Ulis 06/06 09H28	-20.00	6811c5838095a6de2184ce8f845aa68d852d8ba8522449e90d118f5b0fd9df45	30	1	Retrait Distributeur Les Ulis	Ret Les Ulis	
289	7	2026-06-05	Web M. Madi Toumani Ben-moha	-2850.00	a2d9fc28a764971e73db5eaa0a5ba7ab329fa9f666b62730639f283b79b8aa1d	31	1	Vers Livret A	Web M. Madi Toumani Ben-moha	
28	1	2026-01-09	Prlv Acheel France REF-927B2080281923C168GFGWBH4 Core	-18.90	4daf86469e7f882869b03bd4cebd590d9fd08161529330929d87f3fd66b79af9	22	1	Assurance habitation Acheel France	Acheel France Core	
294	7	2026-06-08	X4519 Viandes Du 95 Argent 06/06	-7.50	82ae17ab488d668147c9f5a71a2c6eaa7dd4bdd670080b1c30ff0596deea926d	24	1	Boucherie Argenteuil	Viandes Du Argent	
198	4	2026-04-14	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-15.97	1ffe2c0cd3c623b0668bb41b1ff3f172a8ff7e58060627b82dc5889b84131799	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
306	7	2026-06-12	Prlv Bouygues Telecom 06xxxxx951BT1150PNIAM30 Core	-13.99	84a561d9edbc429a287ca123c1a5213c67c0b08830d0c032b313e66b36671221	32	1	Bouygues Telecom Mobile	Bouygues Telecom	
23	1	2026-01-07	Prlv Free Telecom Free Hautdebit 144160875547939382 Core	-19.99	80da31ab0823bea7678f818fc9030042517bdea30aef68cc50b9e492530395e0	22	1	Free	Free	
163	4	2026-03-24	Carte X4519 Amazon Payments Payl 23/03	-27.98	942bdb6c0e2622ab0e0f429e43c6ebb8067a8eeeba4d8e9deec6f080a1ce91dd	26	1	Amazon	Amazon	
50	2	2026-01-22	X4519 Heetch* Transport Ba 21/01	-18.00	9da66a6190075089a8560cce0f06e36145d8c8443669d01a0d7d6169f8c21ca6	23	1	Heetch	Heetch Transport Ba	
83	2	2026-02-04	Carte X4519 Royal Hotel Monrovia 26/01	-46.49	25679694dd4ea7552feedf083cda6e0380e63e7fdcca87fa8e6ac11437bbbe38	29	1	Royal Hotel Monrovia	Royal Hotel Monrovia	
41	1	2026-01-17	Virement Vir Inst vers Ben-mohamed Madi T	-500.00	4be15193150936f1ea129e20de9fde2a44e18ea37762450d84036ec4894d6d49	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
57	2	2026-01-24	Virement Vir Inst vers Ben-mohamed Madi T	-120.00	8bc65e8f13143d4f44556dfa17f050f386fbc6339bd586b730bfcafec02d63c3	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
64	2	2026-01-21	Virement Vir Inst vers Ben-mohamed Madi T	-280.00	53e5f3e8b5907cd3c8837e7e623a2f9b421aa79d89ecebd96ae6a606ff79211f	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
93	2	2026-02-12	Vir Inst vers Ben-mohamed Madi T	-91.00	6a17215a46b78a98a04184beb3671a9912e885230975b07e5829a25c6b57bb48	25	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
67	2	2026-01-29	Vir Inst vers Ben-mohamed Madi T	-100.00	d4ba6ad37c4209dd295b2b5428f2116c382a1ce1d2352eb54ad35b70e7bef087	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
79	2	2026-02-03	Vir Inst vers Ben-mohamed Madi T	-450.00	2d14bb2ed0895d6d0e90cb5c471d183e3ec1b17eb02000e9b3670ba0922a9647	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
87	2	2026-02-08	Vir Inst vers Ben-mohamed Madi T	-100.00	0fae0d56dac69a70c4741d79c9c8d7f4e229dc51df3367219b57f774ea038106	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
91	2	2026-02-10	Vir Inst vers Ben-mohamed Madi T	-180.00	e9bb5c4389204e37963eb2114a61a65e049998c16918ed42a90b04692e179713	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
107	2	2026-02-18	Vir Inst vers Ben-mohamed Madi T	-30.00	bffffc774f1193185d93dc3a1d48454bcd6b00558fa6177437b7e556096d90f8	30	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
95	2	2026-02-15	Vir Inst vers Ben-mohamed Madi T	-42.00	fe231ea8592f2bd97aeb28469fe4fb52338ff8eb9ea9f7f8fe9b2349ad0816fe	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
113	2	2026-02-20	Vir Inst vers Ben-mohamed Madi T	-30.00	2dc849fa30075af0d70dd46bbe21f9c22eeeaa93e4a6c8aaf27bba7da80d7288	27	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
114	3	2026-02-23	Virement Vir Inst vers Ben-mohamed Madi T	-20.00	ef1225328026d19f6fd4f5b5b2bb6825206bb45022b35e47952ea7bf8cbecb74	27	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
156	4	2026-03-22	Vir Inst vers Ben-mohamed Madi T	-26.00	65174ba330f104f5aa61a8c5560abeb6a6ae85a13d78164e717a5c24d39270a9	27	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
122	3	2026-02-28	Vir Inst vers Ben-mohamed Madi T	-20.00	11da7999aa4bbabce47de6aa9e3b24af3a3962d1d8533a706e3e7c67134b6af8	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
125	3	2026-03-02	Vir Inst vers Ben-mohamed Madi T	-250.00	8893ba636aad2850fd296545a7682bcbee6c4f6eb1e793d5087ef2319118daf5	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
131	3	2026-03-06	Virement Vir Inst vers Ben-mohamed Madi T	-105.00	8d4c48d240a207bb80b7cd5629f0e1bb72f8673d11385aa84770add314c4ea10	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
137	3	2026-03-10	Virement Vir Inst vers Ben-mohamed Madi T	-30.00	3518bf0190783499552487d438d91adfd60d40d7fa15058ff3de0c16bc415fb2	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
149	3	2026-03-17	Vir Inst vers Ben-mohamed Madi T	-49.00	99f1a3a987b7fd0bb15d274e13046eb17c7fb094f679a3db73546749c8f51d70	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
153	3	2026-03-20	Vir Inst vers Ben-mohamed Madi T	-30.00	13f6869849f9e4e67a44789568562a78e53694843579d49fd69e3dcbe39d7d6f	27	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
150	3	2026-03-18	Prlv Engie Mandat Core	-54.88	a77146fb92bae905d5acb42652eb6b7409a0b44c084c8725cad28afb44abc05e	22	1	Engie	Engie	
162	4	2026-03-23	Vir Inst vers Ben-mohamed Madi T	-30.00	065dc62f4a7c4ce8467adcbef45ff82e6b65c705a35c00949ca51fa96ecfeb94	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
172	4	2026-03-31	Vir Inst vers Ben-mohamed Madi T	-50.00	5078a39526a3b4f216ce9c81f9ec25f3a773ca12ab17758709562f2e032bca8f	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
287	7	2026-06-05	Vir Inst vers M.madi Tarik	-300.00	5e0ab6c1e8e4641ca0285c8de9415d38e93dfaa32daed560c87875d26b2d2e44	29	1	Vers Tarik	Vers M.madi Tarik	
290	7	2026-06-06	Virement Vir Inst vers Aya N’goran	-45.00	ce70bb00bc6d888ed4a4a778b2b6a571fc3697ef402816ce2720b1ef69e18878	33	1	Vers Sonia	Vers Aya N’goran	
278	7	2026-06-04	Prlv Acheel France REF-996928EF59FA47A80BGFGWBH4 FR4527787F911 Core	-12.47	3ba6a9b63eb92b5bc1d8d0832321a5f701a9723785544c19327c0d564ceb4dce	22	1	Assurance habitation Acheel France	Acheel France Core	
173	4	2026-04-01	Virement Lessonsharing MTB-2026-03-31F:1-19-01-1	1210.00	1607f6f715f6160493ac6f64bdcf5cc83ec507b562facb30309fd2dc8f9001f0	28	1	LessonSharing	Lessonsharing - - -1	
65	2	2026-01-28	Vir Inst vers Ben-mohamed Madi T	-25.00	5fd1ac8a39b9ca7719b69c0bad51a8351458cba8d701a5c8582b88afc2fe8e87	27	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
185	4	2026-04-07	Carte X4519 Deliveroo 75009 Pari 06/04	-17.33	bc03b58722727e16313be42d36a7a44a4a39636d294238e7b0fef8af46f6814b	24	1	Deliveroo	Deliveroo	
220	5	2026-05-05	Prlv Acheel France REF-81821F2DAF9EDES013GFGWBH4 Core	-12.47	8461738f9a05aca0ec6c046b170eafff5a9463e6cb65ba36f752d6424b308d21	22	1	Assurance habitation Acheel France	Acheel France Core	
305	7	2026-06-12	Carte X4519 Tiktok Shop Seller D 12/06	-12.38	cff73d34a39e074597deebf095e3ed9afdccafd1eeda3f64199c678a68d1dcd6	29	1	Tiktok Shop	Tiktok Shop Seller D	
201	4	2026-04-17	Prlv Engie Mandat Core	-54.88	c04021eb97430308e27b15573eef3c6f6ecccfb6d39f8c12b5161094bdf7614d	22	1	Engie	Engie	
164	4	2026-03-24	Prlv Paypal (europe) $.a r.l. et Cie, 1049085182258 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-162.50	a6477adcc06769ff91f216f650ea9c2d1e9c6e6a6f872aac874c50a9c4b09b52	30	1	Paypal	PayPal	
181	4	2026-04-03	Vir Inst vers Ben-mohamed Madi T	-50.00	475d780a0c04f49d9ecb33f99bbee5d0e385e01111421b041e8123341115b632	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
186	4	2026-04-07	Vir Inst vers Ben-mohamed Madi T	-50.00	65cfa3af9a6f7104bda1ed8bb4ff693989389a197738cfd5c7394203401ae932	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
193	4	2026-04-11	Vir Inst vers Ben-mohamed Madi T	-45.00	e2aede1618600129be31184e482d6565c33e451ebc7d954b978669d11af6d682	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
216	5	2026-05-01	Vir Inst vers Ben-mohamed Madi T	-25.00	b6550cc1ffc04da5457239c79e028377d965ceef03fa2f001e3b12a352d124ef	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
45	1	2026-01-19	Vir Inst vers Ben-mohamed Madi T	-53.00	78980e08109702ffb58365039bbabf79021159b5aeb19fb2f68750390e87c83e	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
301	7	2026-06-10	Prlv Paypal (europe) $.a r.l. et Cie, 1050858593549 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-287.09	dd400ff021a7fda89d8e48f575986f2f48947b4cdd90cd19f079d43c383434d2	26	1	Paypal	PayPal	
29	1	2026-01-09	Vir Inst vers Ben-mohamed Madi T	-650.00	50455fbe38032fdec4de80a5ba5ccb02a7f7c5bb24546e47c00dc52d9d4dccb8	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
286	7	2026-06-05	Vir Inst vers Ben-mohamed Madi T	-354.00	46d6b8376a1e3edfae974cd4d80646858a47c22bea9b0e89710ece6c27baf91c	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
288	7	2026-06-05	Vir Inst vers Ben-mohamed Madi T	-1100.00	cf256175c0d586ef4b4c6dba0013b5e1422acd096b2713bac6121e8f56949815	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
302	7	2026-06-10	Virement Vir Inst vers Ben-mohamed Madi T	-14.00	f6367cc71a5863274517d569ac92e53ab4264fcecc195c7c0b2f9eb09b388c1b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
63	2	2026-01-26	Vir Inst vers Ben-mohamed Madi T	-52.00	e728d6d200cc60f6ac34e36a001c78937162a257b3390178885ba28d36e655f9	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
89	2	2026-02-09	Vir Inst vers Ben-mohamed Madi T	-100.00	843fe0b254991120868056893249e570447b42544eb6d79aa52ddc71037d508b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
94	2	2026-02-13	Vir Inst vers Ben-mohamed Madi T	-80.00	697ca8d4f45b52c3ab59cf0296f468a4619bb4f081b7bb16706cac87f244760b	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
109	2	2026-02-19	Vir Inst vers Ben-mohamed Madi T	-720.00	2f18eadc8faa829d7ff2e46e388c9a87b49cc1aa2bd438cadb98d46116ec2237	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
112	2	2026-02-20	Virement Vir Inst vers Ben-mohamed Madi T	-970.00	f137e26ee024e7051eb0f734990da95ea56b06aa400f7040a16bc7d3efbb2446	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
117	3	2026-02-25	Vir Inst vers Ben-mohamed Madi T	-20.00	19ed22690b98a4a802649dc9cf5c1b3a99ad4a1e502e7a657586fc64254799dd	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
135	3	2026-03-09	Vir Inst vers Ben-mohamed Madi T	-110.00	44c89b7edd495f222b46b4c1f54e775fd2d63be352438b4bcd563534579aa821	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
138	3	2026-03-11	Vir Inst vers Ben-mohamed Madi T	-50.00	9b8ec3259926834c9336672d1605c4b1f01ecc89f00ae492c97e06a1fd187147	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
144	3	2026-03-14	Vir Inst vers Ben-mohamed Madi T	-50.00	34452a9406513e50302412c8d29b6a43904d96f0f2cafe3bf1821d76485f14d3	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
169	4	2026-03-28	Vir Inst vers Ben-mohamed Madi T	-380.00	0dd1d7980c18798bd907d66017c99f63380833eb51ee2b10cf18542f28c116dd	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
309	7	2026-06-15	Virement Vir Inst vers Ben-mohamed Madi T	-605.00	e606650e849cea3b6ffee439ee5c43c40aba2be038b52d2d01e64dbd72f5ce37	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
111	2	2026-02-20	De M. Madi Toumani Ben-moh	30.00	7ad9e59b96fbaba88523ec0e572fd9fdb792f0b0dfd8a2fd383c2cbe13e2b079	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
179	4	2026-04-03	De M. Madi Toumani Ben-moh	1850.00	ee6a0a2160a208cc640abea629b19b2a1c3c6fcede04b3fc806a2228a1a35424	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
213	5	2026-04-29	Virement Vir Inst de Paypal	241.25	abbd4c416140574211add26743213391010e1508091719e7b72d7f717385f7d6	31	1	Paypal	PayPal	
217	5	2026-05-02	Virement Vir Inst de Paypal	162.50	d8c0af2a6ca12cc7f471aa89f461550113e4aa01f186e91ccfba75cb60d85336	31	1	Paypal	PayPal	
214	5	2026-04-30	Prlv Paypal (europe) $.a r.l. et Cie, 1049930192044 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-195.95	b042a44e79c8a47cd911f13cecef5392890bc295f9a56c9ab2929fbf42deee13	26	1	Paypal	PayPal	
56	2	2026-01-23	Prlv Paypal (europe) $.a r.l. et Cie, 1047754968077 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-162.50	e16160ad3d5825b75ceecb0471310a327577a69e5a30a47f19475f35f3cde1b8	26	1	Paypal	PayPal	
197	4	2026-04-13	Prlv Paypal (europe) $.a r.l. et Cie, 1049504421276 Paypal567J2253W2NHALU96ZZZ0000000000000000058 Core	-195.95	a05c794202f2880e4004b4ae7a7dfbec12e36a0b6c0208bc99e7edbb0b180194	29	1	Paypal	PayPal	
281	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00005F:1-17-04-1	1520.00	762c5be0493b8f4fb4c0d5a21ef95beb7425735eecdd9dd1231eb85239f887be	28	1	LessonSharing	Lessonsharing - - - -1	
272	7	2026-05-22	Prlv Idfm IDFM/002073317-4326232-2073281/MAND MANDATE-REFERENCE00000000000055687/7FR42277457385 Core	-20.00	b1bf035f7ccf25d81f4ce7e2bbb7725c64ef48d528daf97ad5da4ef58a527e9c	23	1	IDFM Core	Idfm Idfm - - Mand Mandate- Core	
273	7	2026-05-23	Virement Vir Inst vers Ben-mohamed Madi T	-22.00	3eec624ce043df646a7a82edd5cdf1d7eb1006ecb67dbef83c699543265ad80f	27	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
208	5	2026-04-22	Prlv Paypal (europe) $S.a r.l. et Cie, 1049757239770 Paypal 567J2253W2NHA LU96777Z0000000000000000058 Core	-21.25	b6a2997e79492c20f4a480d1297d6d641c7b807634876eb03da64c381ff92218	26	1	Paypal	PayPal	
228	5	2026-05-20	Carte X4519 Cofidis Amazon Ville 07/05	-63.68	907bde8a4099efe42e8924859bda8ecaebad1a92b7b1e3efe8d46bf1b19cedc4	29	1	Amazon	Amazon	
178	4	2026-04-02	Vir Inst vers Ben-mohamed Madi T	-50.00	f2c60d1ea369d5662d30d3798c43c9620f499fe486893e2219b61850656c2d7e	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
76	2	2026-02-03	Lessonsharing MTB-2026-01-30F:1-22-11-1	620.00	8cbd785650c0d74107e36d10af4611c88c095e309388a6f2dc2518072e077a02	28	1	LessonSharing	Lessonsharing - - -1	
74	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-02-12-1	100.00	8485edce56725fdca0c4c4c4f460d158f08006fc21e3294140d63bc2f17499c4	28	1	LessonSharing	Lessonsharing - - -1	
124	3	2026-03-02	Virement Lessonsharing MTB-2026-02-27F:1-12-01-1	1400.00	c55b1538dbdb7159ea09fe19f4657121300438659a24a81b0175f70736886834	28	1	LessonSharing	Lessonsharing - - -1	
69	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-12-12-1	1120.00	3bf7ad0d49aa12afb2631cbb312649f45c638fb3b78bbef273c0decc88188c5c	28	1	LessonSharing	Lessonsharing - - -1	
9	1	2025-12-29	Virement Vir Inst vers Ben-mohamed Madi T	-660.00	e247cc582c7f8df788a3122cb497b496362dfea5e63f8e6bc887d8f529af036e	29	1	Vers Trade Republic	Vers Ben-mohamed Madi T	
97	2	2026-02-17	Virement De M. Madi Toumani Ben-moh	250.00	08d15a6999b83aac4b0a65e381bebbfbc4f2d46842079730a807cfd5b57bd020	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
78	2	2026-02-03	Lessonsharing MTB-2026-01-30F:1-10-11-1	230.00	34e48e937b45fe10e38a4d3ef75541675a3f0f21b34f471481ae6e9281ad01be	28	1	LessonSharing	Lessonsharing - - -1	
204	4	2026-04-20	Carte X4519 Burger King Les Ulis 18/04	-12.95	6457babaf668fb80445de9d3e30f1e29ecdfcbe93aee14b4b72c57f5cb750963	24	1	Burger King	Burger King	
282	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00002F:1-23-03-1	840.00	482a9a0bd04cb297d681136d203bfc4c76108a23d3e4706ed43fd71180cfa04d	28	1	LessonSharing	Lessonsharing - - - -1	
134	3	2026-03-09	Carte X4519 Deliveroo 75009 Pari 06/03	-25.30	0fe6caa612915c190ac3296281aa6345ff564b4b786668a4ebb98c9488d9fed1	24	1	Deliveroo	Deliveroo	
115	3	2026-02-24	Carte X4519 Yamanashi Paris 23/02	-18.50	780217087928c12cce080b666bc6538eeabdbb00bb8f3f4791f0daa4c54ba92d	28	1	Yamanashi Paris	Yamanashi Paris	
224	5	2026-05-11	X4519 Les Uli 07/05	-8.03	67b2eea7c089e8d51c81a4ef88470a90f3987c433b198a3e895e33207751f37a	24	1	Carrefour les Ulis	Les Uli	
280	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00003F:1-26-03-1	840.00	33b68fda05ed7ed46ef671d4656513eeb987332a6a1a3068a6a951c7b63134b3	28	1	LessonSharing	Lessonsharing - - - -1	
283	7	2026-06-05	Virement Lessonsharing 2026-06-04-MTB 00006F:1-24-04-1	840.00	ef213fef5ae6785a2384d5d9790e3f5a0c4ea484fb6ab326975c1196f3594d78	28	1	LessonSharing	Lessonsharing - - - -1	
206	4	2026-04-20	Prlv Idfm IDFM/001788508-3515435-1788199/MAND ATE-REFERENCE000000000000556877 Core	-6.12	d75626e5f0bb7736df2bc18ad43b4aeca696d77c06f92bb6981fa1ad023d1699	23	1	IDFM Core	Idfm Idfm - - Mand Ate- Core	
72	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-17-11-1	1210.00	08e510e954adc3cb54a6ffb32d507353304685d92db5ad9691a833b5044fc25f	28	1	LessonSharing	Lessonsharing - - -1	
73	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-01-11-1	920.00	da008674f141a826aea2db4f15fe7ec4573fb0a4089e0ffe8b1c25f260bc7db8	28	1	LessonSharing	Lessonsharing - - -1	
75	2	2026-02-03	Virement Lessonsharing MTB-2026-01-30F:1-27-10-1	1520.00	d765dd445a67674637d15baf2470e85755fe2796b5cef60c50d7b9e319f29904	28	1	LessonSharing	Lessonsharing - - -1	
308	7	2026-06-15	De M. Madi Toumani Ben-moh	605.00	c08dce2048516d42d084f14b55e65c6e65c4ecb29ed2d7b64f7dcf8479f6dc4c	31	1	Vers Compte courant	De M. Madi Toumani Ben-moh	
2	1	2025-12-24	Virement Lessonsharing MTB-2025-12-22F:1-11-10-1	1520.00	f2713a5dacedcb167cf6f3eb49c8b549bfe3bd4c03aed4e91aac45751d5cbd7e	28	1	LessonSharing	Lessonsharing - - -1	
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: themis
--

COPY public.users (id, username, password_hash, role, created_at) FROM stdin;
1	Ben	pbkdf2$sha256$210000$Nwkgc4OoYUlEzCP8YtTOgQ$wmavo1XXdxoUJ7tvZ+bHMk790ZPgza0rwaXfE2tP8ew	admin	2026-07-18 13:33:46.61631+00
2	localadm	pbkdf2$sha256$210000$RVTjzC7gHflUCBglzShg0Q$a3/fDxHvU27Djexp1eNZt2ZmZNgKNt7HYRyoo9MO5Uw	user	2026-07-18 17:42:32.761622+00
\.


--
-- Name: account_snapshots_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.account_snapshots_id_seq', 8, true);


--
-- Name: accounts_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.accounts_id_seq', 8, true);


--
-- Name: analyses_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.analyses_id_seq', 8, true);


--
-- Name: budgets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.budgets_id_seq', 7, true);


--
-- Name: credits_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.credits_id_seq', 3, true);


--
-- Name: domains_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.domains_id_seq', 43, true);


--
-- Name: merchant_aliases_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.merchant_aliases_id_seq', 83, true);


--
-- Name: rules_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.rules_id_seq', 106, true);


--
-- Name: statements_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.statements_id_seq', 7, true);


--
-- Name: tags_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.tags_id_seq', 1979, true);


--
-- Name: taxes_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.taxes_id_seq', 3, true);


--
-- Name: transactions_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.transactions_id_seq', 312, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: themis
--

SELECT pg_catalog.setval('public.users_id_seq', 2, true);


--
-- Name: account_snapshots account_snapshots_account_id_snap_date_key; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots
    ADD CONSTRAINT account_snapshots_account_id_snap_date_key UNIQUE (account_id, snap_date);


--
-- Name: account_snapshots account_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots
    ADD CONSTRAINT account_snapshots_pkey PRIMARY KEY (id);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: analyses analyses_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.analyses
    ADD CONSTRAINT analyses_pkey PRIMARY KEY (id);


--
-- Name: budgets budgets_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_pkey PRIMARY KEY (id);


--
-- Name: credits credits_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.credits
    ADD CONSTRAINT credits_pkey PRIMARY KEY (id);


--
-- Name: domains domains_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT domains_pkey PRIMARY KEY (id);


--
-- Name: merchant_aliases merchant_aliases_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.merchant_aliases
    ADD CONSTRAINT merchant_aliases_pkey PRIMARY KEY (id);


--
-- Name: rules rules_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (name);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (token);


--
-- Name: statements statements_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements
    ADD CONSTRAINT statements_pkey PRIMARY KEY (id);


--
-- Name: tag_folder_prefs tag_folder_prefs_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tag_folder_prefs
    ADD CONSTRAINT tag_folder_prefs_pkey PRIMARY KEY (owner_id, tag_name);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: taxes taxes_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.taxes
    ADD CONSTRAINT taxes_pkey PRIMARY KEY (id);


--
-- Name: transaction_tags transaction_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transaction_tags
    ADD CONSTRAINT transaction_tags_pkey PRIMARY KEY (transaction_id, tag_id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: idx_accounts_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_accounts_owner ON public.accounts USING btree (owner_id);


--
-- Name: idx_accounts_owner_name; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_accounts_owner_name ON public.accounts USING btree (owner_id, name);


--
-- Name: idx_analyses_owner_month; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_analyses_owner_month ON public.analyses USING btree (owner_id, month);


--
-- Name: idx_budgets_owner_domain; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_budgets_owner_domain ON public.budgets USING btree (owner_id, domain_id);


--
-- Name: idx_domains_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_domains_owner ON public.domains USING btree (owner_id);


--
-- Name: idx_domains_owner_name_parent; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_domains_owner_name_parent ON public.domains USING btree (owner_id, name, COALESCE(parent_id, (0)::bigint));


--
-- Name: idx_domains_owner_root_name; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_domains_owner_root_name ON public.domains USING btree (owner_id, name) WHERE (parent_id IS NULL);


--
-- Name: idx_merchant_aliases_owner_key; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_merchant_aliases_owner_key ON public.merchant_aliases USING btree (owner_id, lower(match_key));


--
-- Name: idx_rules_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_rules_owner ON public.rules USING btree (owner_id);


--
-- Name: idx_rules_owner_pattern_ci; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_rules_owner_pattern_ci ON public.rules USING btree (owner_id, lower(pattern));


--
-- Name: idx_sessions_expires; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_sessions_expires ON public.sessions USING btree (expires_at);


--
-- Name: idx_settings_owner_key; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_settings_owner_key ON public.settings USING btree (COALESCE(owner_id, (0)::bigint), key);


--
-- Name: idx_statements_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_statements_owner ON public.statements USING btree (owner_id);


--
-- Name: idx_tags_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_tags_owner ON public.tags USING btree (owner_id);


--
-- Name: idx_tags_owner_name; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_tags_owner_name ON public.tags USING btree (owner_id, name);


--
-- Name: idx_transactions_label; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_transactions_label ON public.transactions USING gin (to_tsvector('simple'::regconfig, label));


--
-- Name: idx_transactions_owner; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_transactions_owner ON public.transactions USING btree (owner_id);


--
-- Name: idx_transactions_owner_hash; Type: INDEX; Schema: public; Owner: themis
--

CREATE UNIQUE INDEX idx_transactions_owner_hash ON public.transactions USING btree (owner_id, hash);


--
-- Name: idx_transactions_owner_merchant; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_transactions_owner_merchant ON public.transactions USING btree (owner_id, merchant);


--
-- Name: idx_tx_date; Type: INDEX; Schema: public; Owner: themis
--

CREATE INDEX idx_tx_date ON public.transactions USING btree (tx_date);


--
-- Name: account_snapshots account_snapshots_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.account_snapshots
    ADD CONSTRAINT account_snapshots_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: accounts accounts_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: analyses analyses_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.analyses
    ADD CONSTRAINT analyses_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: budgets budgets_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.domains(id) ON DELETE CASCADE;


--
-- Name: budgets budgets_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.budgets
    ADD CONSTRAINT budgets_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: credits credits_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.credits
    ADD CONSTRAINT credits_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: domains domains_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT domains_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: domains domains_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT domains_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.domains(id) ON DELETE CASCADE;


--
-- Name: merchant_aliases merchant_aliases_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.merchant_aliases
    ADD CONSTRAINT merchant_aliases_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: rules rules_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.domains(id) ON DELETE CASCADE;


--
-- Name: rules rules_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: settings settings_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: statements statements_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements
    ADD CONSTRAINT statements_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: statements statements_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.statements
    ADD CONSTRAINT statements_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: tag_folder_prefs tag_folder_prefs_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tag_folder_prefs
    ADD CONSTRAINT tag_folder_prefs_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: tags tags_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: taxes taxes_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.taxes
    ADD CONSTRAINT taxes_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: transaction_tags transaction_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transaction_tags
    ADD CONSTRAINT transaction_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: transaction_tags transaction_tags_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transaction_tags
    ADD CONSTRAINT transaction_tags_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES public.transactions(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_domain_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES public.domains(id) ON DELETE SET NULL;


--
-- Name: transactions transactions_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: transactions transactions_statement_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: themis
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_statement_id_fkey FOREIGN KEY (statement_id) REFERENCES public.statements(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict atIFF51PQIho94DmdGMRhbOP31Fs8Va6nb4m6vvhUqWI80ZOnVbrMFhRe5Tex1O

