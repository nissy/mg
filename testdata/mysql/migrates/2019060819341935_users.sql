-- @migrate.up
CREATE TABLE users (
  id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  name varchar(255) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL,
  created_at datetime DEFAULT NULL,
  updated_at datetime DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

-- @migrate.down
DROP TABLE users;