-- ============================================================
--  FACILITY TAGS: the catalogue every platform starts with.
--  Superadministrators edit it at /admin ("Etiquetas"); councils only pick from it.
-- ============================================================

WITH c AS (INSERT INTO tag_category (name, position) VALUES ('Deportes', 1) RETURNING id)
INSERT INTO tag (category_id, slug, name)
SELECT c.id, t.slug, t.name FROM c, (VALUES
    ('futbol', 'Fútbol'),
    ('futbol-sala', 'Fútbol sala'),
    ('futbol-7', 'Fútbol 7'),
    ('baloncesto', 'Baloncesto'),
    ('balonman', 'Balonmán'),
    ('voleibol', 'Voleibol'),
    ('padel', 'Pádel'),
    ('tenis', 'Tenis'),
    ('tenis-de-mesa', 'Tenis de mesa'),
    ('badminton', 'Bádminton'),
    ('squash', 'Squash'),
    ('fronton', 'Frontón'),
    ('natacion', 'Natación'),
    ('waterpolo', 'Waterpolo'),
    ('atletismo', 'Atletismo'),
    ('ximnasia', 'Ximnasia'),
    ('musculacion', 'Musculación'),
    ('actividades-dirixidas', 'Actividades dirixidas'),
    ('artes-marciais', 'Artes marciais'),
    ('escalada', 'Escalada'),
    ('patinaxe', 'Patinaxe'),
    ('hoquei', 'Hóckey'),
    ('rugby', 'Rugby')
) AS t (slug, name);

WITH c AS (INSERT INTO tag_category (name, position) VALUES ('Tipo de instalación', 2) RETURNING id)
INSERT INTO tag (category_id, slug, name)
SELECT c.id, t.slug, t.name FROM c, (VALUES
    ('multiusos', 'Multiusos'),
    ('pavillon', 'Pavillón'),
    ('piscina', 'Piscina'),
    ('pista', 'Pista'),
    ('campo', 'Campo'),
    ('sala', 'Sala')
) AS t (slug, name);

WITH c AS (INSERT INTO tag_category (name, position) VALUES ('Servizos', 3) RETURNING id)
INSERT INTO tag (category_id, slug, name)
SELECT c.id, t.slug, t.name FROM c, (VALUES
    ('cuberta', 'Cuberta'),
    ('aire-libre', 'Aire libre'),
    ('iluminacion', 'Iluminación'),
    ('climatizada', 'Climatizada'),
    ('cesped-artificial', 'Céspede artificial'),
    ('accesible', 'Accesible'),
    ('vestiarios', 'Vestiarios')
) AS t (slug, name);

WITH c AS (INSERT INTO tag_category (name, position) VALUES ('Público', 4) RETURNING id)
INSERT INTO tag (category_id, slug, name)
SELECT c.id, t.slug, t.name FROM c, (VALUES
    ('infantil', 'Infantil'),
    ('xuvenil', 'Xuvenil'),
    ('adultos', 'Adultos'),
    ('maiores', 'Maiores'),
    ('deporte-adaptado', 'Deporte adaptado')
) AS t (slug, name);
