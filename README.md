# 📯 Moist-Von-Lipwig

A pet project inspired by the character Moist von Lipwig from Terry Pratchett's *Discworld*. This system functions as a digital "Time Capsule" or a chaotic post office. Users can send messages in various formats (text, audio, images, etc.) to others or their future selves, with one catch: **the time of delivery is unknown.**

The Post Office **WILL SURELY** deliver the message. Eventually.



---

## Why tho????
This project explores what communication looks like when **immediacy is removed**. Instead of the modern "instant" culture where we expect a follow-up immediately, this is "send and wait without knowing." 

You might receive the access notification the very next moment, or ten years might go by until you've forgotten the message even exists—and then, and only then, does it arrive.

---

## Security & Privacy
Because messages may sit in the "sorting floor" for years, security is paramount. We use a **Zero-Knowledge** approach:

* **Access Verification:** Waybill Keys are never stored in plain text. They are hashed using **Bcrypt** ($2a$ cost).
* **Flexible Recipients:** Uses PostgreSQL **JSONB** arrays to store multiple `WaybillID` and `Key` pairs for a single parcel. This way posts can onlly be accessed if you have at least one matching pair.

---

## Uses 
* **Letters to a Future Self:** Send advice or memories to your future self. Delivery is guaranteed, but the timing remains a surprise.
* **Delayed Connection:** Send a message to a friend under the same conditions—a digital "bottle in the ocean."
---

## Non-Goals
* **Not a replacement for IM:** This is not for modern digital comm.
* **Not optimized for speed:** Efficiency is secondary to the "Surety of Delivery."
* **Not predictable:** The delivery schedule is intentionally opaque.

---
