Answers:
1. 
# Ищем название нужной зоны
```console
timedatectl list-timezones | grep "Moscow"
```
# Ставим и проверяем настройки даты
```console
timedatectl set-timezone "Europe/Moscow"
date
```
![alt text](<1. Set_timezone-1.png>)

# Устанавливаем ntp демон
```console
apt install ntp
```
# Настраиваем конфигурационный файл
```console
vim /etc/ntpsec/ntp.conf
```
# Делаем проверку синхронизации и работы службы
```console
systemctl status ntp
ntpq -p
date
```
![alt text](<1. Set_ntp-1.png>)

2. 
# Открываем конфигурационный файл и вносим нужные изменения
```console
vim /etc/netplan/50-cloud-init.yaml
```
# Провряем и применяем настройки
```console
netplan try
netplan apply
ip a
ping google.com
```
![alt text](<2. Set_netplan-1.png>)

3. 
# Создаем пользователей, проверяем группы и задаем пароль
```console
useradd -m -s /bin/bash user1
useradd -m -s /bin/bash user2
passwd user1
passwd user2
groups user1
groups user2
```
![alt text](<3. Create_users-1.png>)

# Даем доступ группе user1 - 'apt update', а пользователю user2 использовать 'systemctl status'
```console
visudo
visudo -c
```
![alt text](<3. Set_visudo-1.png>)

# Создаем ssh ключи и копируем их на сервер
```console
ssh-keygen -t ed25519 -C "user1"
ssh-keygen -t ed25519 -C "user2"
ssh-copy-id -i "/home/user1/.ssh/id_ed25519.pub" ubuntu@192.168.1.212
ssh-copy-id -i "/home/user2/.ssh/id_ed25519.pub" ubuntu@192.168.1.212
```
![alt text](<3. Create_ssh_user1-1.png>)
![alt text](<3. Create_ssh_user2-1.png>)
![alt text](<3. Add_pubkey_on_server-1.png>)

# Закрываем доступ по логину, даем доступ по ключам
vim /etc/ssh/sshd_config.d/50-cloud-init.conf
(скрин 3. SSH_login_pwd_disable)

4. 
# Создаем папку и даем права наследования
```console
sudo mkdir /home/share
sudo chmod -R g+s /home/share
cd /home/share
mkdir my_folder
chmod 700 /home/share/my_folder
```
![alt text](<4. Check_privilages-1.png>)

5. 
# Настройка доступа к портам и проверка правил + nginx
```console
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
ufw default deny incoming
ufw default allow outgoing
ufw enable
ufw status verbose
apt install nginx
```
![alt text](<5. Nginx-1.png>)

### Изменений никаких нет, так как открыт как раз 80 порт, по которому и открывается дефолтная страница nginx.
### Если запретить 80 порт и 22, то перестанет работать ssh и закроется доступ до бразуерной страницы nginx.