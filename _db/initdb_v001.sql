CREATE TABLE IF NOT EXISTS api_gateway_config
(
    id                INT(8) NOT NULL AUTO_INCREMENT,
    reqrepo           VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
    reqitem           VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
    createuser        VARCHAR(64) NOT NULL,
    reqresourcepage   VARCHAR(255) NOT NULL,
    urlpath           VARCHAR(255) NOT NULL,
    reqhostinfo       VARCHAR(255),
    reqappkey         VARCHAR(255) NOT NULL,
    isverify          INT(2) NOT NULL,
    ishttps           INT(2) NOT NULL,
    querytimes        INT(32) NOT NULL,
    reqtype           INT(2) NOT NULL,
    posttemplate      VARCHAR(1024),
    PRIMARY KEY (id),
    CONSTRAINT `UK_REPO_ITEM` UNIQUE (reqrepo, reqitem)

)  DEFAULT CHARSET=UTF8;

CREATE TABLE IF NOT EXISTS api_param (
  id                  INT(8) NOT NULL AUTO_INCREMENT,
  `name`              VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  must                INT(2) NOT NULL,
  `type`              VARCHAR(64) NOT NULL,
  apiId               INT(8) NOT NULL,
  PRIMARY KEY (id)

)  DEFAULT CHARSET=UTF8;

CREATE TABLE IF NOT EXISTS DF_ITEM_STAT
(
   STAT_KEY     VARCHAR(255) NOT NULL COMMENT '3*255 = 765 < 767',
   STAT_VALUE   INT NOT NULL,
   PRIMARY KEY (STAT_KEY)

)  DEFAULT CHARSET=UTF8;

