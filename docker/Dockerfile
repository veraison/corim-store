FROM ubuntu:24.04 AS corim-store-db

RUN apt-get update  && \
    DEBIAN_FRONTEND=noninteractive apt-get install \
    	--assume-yes \
	--no-install-recommends \
        mariadb-server \
	postgresql \
	supervisor \
	vim \
	&& \
    apt-get clean && \
    apt-get autoremove --assume-yes && \
    rm -rf /var/lib/apt/lists/* /var/tmp/* /tmp*

RUN mkdir -p /var/run/mysqld && chown -R mysql:mysql /var/run/mysqld && \
    mkdir -p /tmp && chmod 1777 /tmp && \
    mariadb-install-db --user=mysql --ldata=/var/lib/mysql && \
    sed -i "s/^bind-address\s*=.*/bind-address = 0.0.0.0/" /etc/mysql/mariadb.conf.d/50-server.cnf

RUN mariadbd-safe --datadir=/var/lib/mysql & \
    sleep 2 && \
    until mariadb -uroot -e "SELECT 1" &>/dev/null; do sleep 1; done && \
    mariadb -uroot <<EOF
CREATE DATABASE corim_store;
GRANT ALL PRIVILEGES ON corim_store.* TO 'store_user'@'%' IDENTIFIED BY 'L3tM31n';
ALTER USER 'store_user'@'%' REQUIRE NONE;
FLUSH PRIVILEGES;
EOF

EXPOSE 3306

USER postgres

RUN /usr/lib/postgresql/16/bin/initdb -D /var/lib/postgresql/data && \
    sed -i "s/^#listen_addresses\s*=.*$/listen_addresses = '*'/" /var/lib/postgresql/data/postgresql.conf && \
    echo "host all all 0.0.0.0/0 scram-sha-256" >> /var/lib/postgresql/data/pg_hba.conf && \
    /usr/lib/postgresql/16/bin/postgres --single -D /var/lib/postgresql/data postgres <<EOF
CREATE DATABASE corim_store;
CREATE USER store_user WITH PASSWORD 'L3tM31n';
GRANT ALL PRIVILEGES ON DATABASE corim_store TO store_user;
EOF

RUN /usr/lib/postgresql/16/bin/postgres --single -D /var/lib/postgresql/data corim_store <<EOF
GRANT ALL ON SCHEMA public TO store_user;
EOF

EXPOSE 5432

USER root

COPY supervisord.conf /etc/supervisor/supervisord.conf

CMD ["/usr/bin/supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
