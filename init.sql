create table if not exists dictionary (id integer primary key generated always as identity, text varchar(50) unique not null, description varchar(256) not null);
