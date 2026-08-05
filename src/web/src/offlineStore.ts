const databaseName = 'jobdeck-offline';
const storeName = 'snapshots';
const databaseVersion = 1;

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = window.indexedDB.open(databaseName, databaseVersion);
    request.addEventListener('upgradeneeded', () => {
      const database = request.result;
      if (!database.objectStoreNames.contains(storeName)) {
        database.createObjectStore(storeName);
      }
    });
    request.addEventListener('success', () => resolve(request.result));
    request.addEventListener('error', () => reject(request.error));
  });
}

export async function readSnapshot<T>(key: string): Promise<T | null> {
  const database = await openDatabase();
  return new Promise((resolve, reject) => {
    const transaction = database.transaction(storeName, 'readonly');
    const request = transaction.objectStore(storeName).get(key);
    request.addEventListener('success', () => resolve((request.result as T | undefined) ?? null));
    request.addEventListener('error', () => reject(request.error));
    transaction.addEventListener('complete', () => database.close());
  });
}

export async function writeSnapshot<T>(key: string, value: T): Promise<void> {
  const database = await openDatabase();
  const serializableValue = JSON.parse(JSON.stringify(value)) as T;
  await new Promise<void>((resolve, reject) => {
    const transaction = database.transaction(storeName, 'readwrite');
    transaction.objectStore(storeName).put(serializableValue, key);
    transaction.addEventListener('complete', () => resolve());
    transaction.addEventListener('error', () => reject(transaction.error));
  });
  database.close();
}
