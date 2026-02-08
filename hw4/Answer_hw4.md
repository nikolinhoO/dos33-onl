Answers:
1. 
# Ищем название нужной зоны
timedatectl list-timezones | grep "Moscow" -
# Ставим и проверяем настройки даты
timedatectl set-timezone "Europe/Moscow"
date
(скрин 1. Set_timezone)

# Устанавливаем ntp демон
apt install ntp
# Настраиваем конфигурационный файл
vim /etc/ntpsec/ntp.conf
# Делаем проверку синхронизации и работы службы
systemctl status ntp
ntpq -p
date
(скрин 1. Set_ntp)

2. 
# Открываем конфигурационный файл и вносим нужные изменения
vim /etc/netplan/50-cloud-init.yaml
# Провряем и применяем настройки
netplan try
netplan apply
ip a
ping google.com
(скрин 2. Set_netplan)

3. 
# Создаем пользователей, проверяем группы и задаем пароль
useradd -m -s /bin/bash user1
useradd -m -s /bin/bash user2
passwd user1
passwd user2
groups user1
groups user2
(скрин 3. Create_users)

# Даем доступ группе user1 - 'apt update', а пользователю user2 использовать 'systemctl status'
visudo
visudo -c
(скрин 3. Set_visudo)

# Создаем ssh ключи и копируем их на сервер
ssh-keygen -t ed25519 -C "user1"
ssh-keygen -t ed25519 -C "user2"
ssh-copy-id -i "/home/user1/.ssh/id_ed25519.pub" ubuntu@192.168.1.212
ssh-copy-id -i "/home/user2/.ssh/id_ed25519.pub" ubuntu@192.168.1.212
(скрины 3. Create_ssh_user1, 3. Create_ssh_user2, 3. Add_pubkey_on_server)

# Закрываем доступ по логину, даем доступ по ключам
vim /etc/ssh/sshd_config.d/50-cloud-init.conf
(скрин 3. SSH_login_pwd_disable)

4. 
# Создаем папку и даем права наследования
sudo mkdir /home/share
sudo chmod -R g+s /home/share
cd /home/share
mkdir my_folder
chmod 700 /home/share/my_folder
(скрин 4. Check_privilages)

5. 
# Настройка доступа к портам и проверка правил + nginx
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
ufw default deny incoming
ufw default allow outgoing
ufw enable
ufw status verbose
apt install nginx
(скрин 5. Nginx)

### Изменений никаких нет, так как открыт как раз 80 порт, по которому и открывается дефолтная страница nginx.
### Если запретить 80 порт и 22, то перестанет работать ssh и закроется доступ до бразуерной страницы nginx.