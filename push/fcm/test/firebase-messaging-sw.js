// Firebase Messaging Service Worker
// Поддержка динамической конфигурации

let app = null;
let messaging = null;

self.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'CONFIG') {
        const config = event.data.config;
        console.log('[SW] Received config for project:', config.projectId);

        try {
            importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-app-compat.js');
            importScripts('https://www.gstatic.com/firebasejs/10.7.1/firebase-messaging-compat.js');

            if (!app) {
                app = firebase.initializeApp(config);
                messaging = firebase.messaging();
                console.log('[SW] Firebase initialized');
            }
        } catch (error) {
            console.error('[SW] Init error:', error);
        }
    }
});

// Handle background messages
self.addEventListener('push', (event) => {
    if (!event.data) return;

    const data = event.data.json();
    console.log('[SW] Push received:', data);

    const notificationTitle = data.notification?.title || 'Notification';
    const notificationOptions = {
        body: data.notification?.body || 'New notification',
        icon: data.notification?.icon || '🔔',
        badge: '🔔',
        data: data.data
    };

    event.waitUntil(
        self.registration.showNotification(notificationTitle, notificationOptions)
    );
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    event.waitUntil(clients.openWindow('/'));
});
