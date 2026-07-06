const { onDocumentCreated } = require("firebase-functions/v2/firestore");
const admin = require("firebase-admin");

admin.initializeApp();

exports.onMessageCreated = onDocumentCreated("threads/{threadId}/messages/{messageId}", async (event) => {
  const snapshot = event.data;
  if (!snapshot) {
    console.log("No data associated with the event");
    return;
  }

  const message = snapshot.data();
  const threadId = event.params.threadId;

  console.log(`New message created in thread ${threadId}: ${message.id}`);

  const senderId = message.senderId;
  const content = message.content;

  // 1. Fetch the thread document to get members
  const threadDoc = await admin.firestore().collection("threads").doc(threadId).get();
  if (!threadDoc.exists) {
    console.log(`Thread ${threadId} does not exist`);
    return;
  }

  const thread = threadDoc.data();
  const members = thread.members || [];
  const threadType = thread.type || "DIRECT";

  // 2. Resolve recipients (all members except sender)
  const recipients = members.filter(uid => uid !== senderId);
  if (recipients.length === 0) {
    console.log("No recipients to notify");
    return;
  }

  // Determine sender display name
  const senderName = (thread.buyerId === senderId) 
    ? (thread.buyerName || "Buyer") 
    : (thread.sellerName || "Seller");

  console.log(`Sending notifications to recipients: ${recipients.join(", ")}`);

  // 3. For each recipient, fetch their device tokens
  for (const recipientId of recipients) {
    const tokensSnapshot = await admin.firestore()
      .collection("user_device_tokens")
      .where("userId", "==", recipientId)
      .get();

    if (tokensSnapshot.empty) {
      console.log(`No device tokens registered for user ${recipientId}`);
      continue;
    }

    const tokens = [];
    tokensSnapshot.forEach(doc => {
      const data = doc.data();
      if (data.token) {
        tokens.push(data.token);
      }
    });

    if (tokens.length === 0) {
      continue;
    }

    // 4. Build FCM payload
    const payload = {
      notification: {
        title: threadType === "GROUP" || threadType === "OFFICIAL" 
          ? `${senderName} (${thread.sellerName})` 
          : senderName,
        body: content
      },
      data: {
        threadId: threadId,
        click_action: "FLUTTER_NOTIFICATION_CLICK"
      }
    };

    // 5. Send multicast FCM notification
    try {
      const response = await admin.messaging().sendEachForMulticast({
        tokens: tokens,
        notification: payload.notification,
        data: payload.data,
        android: {
          priority: "high",
          notification: {
            channelId: "wemall_notifications",
            sound: "default"
          }
        }
      });
      console.log(`Successfully sent ${response.successCount} messages to user ${recipientId}`);

      // Prune expired/invalid tokens
      if (response.failureCount > 0) {
        const tokensToRemove = [];
        response.responses.forEach((resp, idx) => {
          if (!resp.success) {
            const code = resp.error?.code;
            if (code === 'messaging/invalid-registration-token' || code === 'messaging/registration-token-not-registered') {
              tokensToRemove.push(tokens[idx]);
            }
          }
        });

        for (const token of tokensToRemove) {
          await admin.firestore()
            .collection("user_device_tokens")
            .doc(token)
            .delete();
          console.log(`Pruned invalid token ${token}`);
        }
      }
    } catch (error) {
      console.error(`Error sending message to user ${recipientId}:`, error);
    }
  }
});
