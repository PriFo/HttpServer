#!/usr/bin/env python3
"""
Система уведомлений для Chaos Monkey тестов
Отправка уведомлений о результатах тестов
"""

import json
import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from pathlib import Path
from typing import Dict, List, Optional
from datetime import datetime


class Notifier:
    """Базовый класс для уведомлений"""
    
    def __init__(self, config: Optional[Dict] = None):
        self.config = config or {}
    
    def send(self, subject: str, message: str, results: Optional[Dict] = None) -> bool:
        """Отправка уведомления"""
        raise NotImplementedError


class EmailNotifier(Notifier):
    """Email уведомления"""
    
    def __init__(self, config: Dict):
        super().__init__(config)
        self.smtp_server = config.get('smtp_server', 'smtp.gmail.com')
        self.smtp_port = config.get('smtp_port', 587)
        self.username = config.get('username', '')
        self.password = config.get('password', '')
        self.from_email = config.get('from_email', '')
        self.to_emails = config.get('to_emails', [])
    
    def send(self, subject: str, message: str, results: Optional[Dict] = None) -> bool:
        """Отправка email уведомления"""
        if not self.to_emails:
            return False
        
        try:
            msg = MIMEMultipart()
            msg['From'] = self.from_email
            msg['To'] = ', '.join(self.to_emails)
            msg['Subject'] = subject
            
            # Формирование тела письма
            body = message
            if results:
                body += "\n\nРезультаты тестов:\n"
                for test_name, passed in results.items():
                    status = "✅ PASSED" if passed else "❌ FAILED"
                    body += f"  - {test_name}: {status}\n"
            
            msg.attach(MIMEText(body, 'plain', 'utf-8'))
            
            # Отправка
            server = smtplib.SMTP(self.smtp_server, self.smtp_port)
            server.starttls()
            server.login(self.username, self.password)
            server.send_message(msg)
            server.quit()
            
            return True
        except Exception as e:
            print(f"Error sending email: {e}")
            return False


class FileNotifier(Notifier):
    """Уведомления в файл"""
    
    def __init__(self, config: Dict):
        super().__init__(config)
        self.notifications_file = Path(config.get('notifications_file', './notifications.log'))
        self.notifications_file.parent.mkdir(parents=True, exist_ok=True)
    
    def send(self, subject: str, message: str, results: Optional[Dict] = None) -> bool:
        """Запись уведомления в файл"""
        try:
            timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            notification = {
                'timestamp': timestamp,
                'subject': subject,
                'message': message,
                'results': results
            }
            
            # Добавляем в файл
            notifications = []
            if self.notifications_file.exists():
                try:
                    notifications = json.loads(self.notifications_file.read_text(encoding='utf-8'))
                except:
                    notifications = []
            
            notifications.append(notification)
            
            # Сохраняем только последние 100 уведомлений
            notifications = notifications[-100:]
            
            self.notifications_file.write_text(
                json.dumps(notifications, indent=2, ensure_ascii=False),
                encoding='utf-8'
            )
            
            return True
        except Exception as e:
            print(f"Error writing notification: {e}")
            return False


class ConsoleNotifier(Notifier):
    """Уведомления в консоль"""
    
    def send(self, subject: str, message: str, results: Optional[Dict] = None) -> bool:
        """Вывод уведомления в консоль"""
        print("\n" + "=" * 60)
        print(f"📢 {subject}")
        print("=" * 60)
        print(message)
        
        if results:
            print("\nРезультаты тестов:")
            for test_name, passed in results.items():
                status = "✅ PASSED" if passed else "❌ FAILED"
                print(f"  {status} {test_name}")
        
        print("=" * 60 + "\n")
        return True


def create_notifier(notifier_type: str, config: Optional[Dict] = None) -> Notifier:
    """Создание уведомления по типу"""
    config = config or {}
    
    if notifier_type == 'email':
        return EmailNotifier(config)
    elif notifier_type == 'file':
        return FileNotifier(config)
    elif notifier_type == 'console':
        return ConsoleNotifier()
    else:
        raise ValueError(f"Unknown notifier type: {notifier_type}")


def send_test_results(results: Dict, notifiers: List[Notifier]):
    """Отправка результатов тестов через все уведомления"""
    total = len(results)
    passed = sum(1 for v in results.values() if v)
    failed = total - passed
    
    subject = f"Chaos Monkey Tests: {passed}/{total} passed"
    message = f"""
Результаты Chaos Monkey тестов

Дата: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}
Всего тестов: {total}
Пройдено: {passed}
Провалено: {failed}
"""
    
    for notifier in notifiers:
        notifier.send(subject, message, results)

