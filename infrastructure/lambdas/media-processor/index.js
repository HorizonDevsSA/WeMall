const { S3Client, GetObjectCommand, PutObjectCommand } = require('@aws-sdk/client-s3');
const sharp = require('sharp');

const s3 = new S3Client({});
const DEST_PUBLIC_BUCKET = process.env.DEST_PUBLIC_BUCKET;
const API_CALLBACK_URL = process.env.API_CALLBACK_URL;

const SIZES = {
  thumbnail_small: { width: 100, height: 100, fit: 'cover' },
  thumbnail_large: { width: 250, height: 250, fit: 'cover' },
  main_mobile: { width: 480, fit: 'inside' },
  main_tablet: { width: 800, fit: 'inside' },
  main_desktop: { width: 1200, fit: 'inside' },
  main_large_retina: { width: 2000, fit: 'inside' }
};

exports.handler = async (event) => {
  console.log("Event:", JSON.stringify(event, null, 2));

  for (const record of event.Records) {
    const srcBucket = record.s3.bucket.name;
    const srcKey = decodeURIComponent(record.s3.object.key.replace(/\+/g, " "));
    
    // e.g. srcKey: uploads/mediaId/filename.jpg
    const parts = srcKey.split('/');
    if (parts.length < 3) continue;
    const mediaId = parts[1];
    const originalFilename = parts[2];

    try {
      // 1. Fetch from S3 Raw
      const getObjectParams = { Bucket: srcBucket, Key: srcKey };
      const s3Object = await s3.send(new GetObjectCommand(getObjectParams));
      const imageBuffer = await streamToBuffer(s3Object.Body);
      const contentType = s3Object.ContentType || '';

      // 2. Generate Variants
      let variantsMap = {};

      if (contentType.startsWith('image/')) {
        const baseImage = sharp(imageBuffer).rotate().withMetadata(false);

        for (const [variantName, resizeOpts] of Object.entries(SIZES)) {
          const resized = baseImage.clone().resize(resizeOpts.width, resizeOpts.height, { fit: resizeOpts.fit || 'inside' });

          // AVIF
          const avifBuffer = await resized.avif({ quality: 65 }).toBuffer();
          const avifKey = `uploads/${mediaId}/${variantName}.avif`;
          await s3.send(new PutObjectCommand({
            Bucket: DEST_PUBLIC_BUCKET,
            Key: avifKey,
            Body: avifBuffer,
            ContentType: 'image/avif',
            CacheControl: 'public, max-age=31536000, immutable'
          }));
          variantsMap[`${variantName}_avif`] = avifKey;

          // WebP
          const webpBuffer = await resized.webp({ quality: 75 }).toBuffer();
          const webpKey = `uploads/${mediaId}/${variantName}.webp`;
          await s3.send(new PutObjectCommand({
            Bucket: DEST_PUBLIC_BUCKET,
            Key: webpKey,
            Body: webpBuffer,
            ContentType: 'image/webp',
            CacheControl: 'public, max-age=31536000, immutable'
          }));
          variantsMap[`${variantName}_webp`] = webpKey;
        }
      } else {
        // Non-image file, just copy it to the public bucket
        const docKey = `uploads/${mediaId}/${originalFilename}`;
        await s3.send(new PutObjectCommand({
          Bucket: DEST_PUBLIC_BUCKET,
          Key: docKey,
          Body: imageBuffer,
          ContentType: contentType,
          CacheControl: 'public, max-age=31536000, immutable'
        }));
        variantsMap['original_url'] = docKey;
      }

      console.log(`Successfully processed variants for ${mediaId}`);

      // 3. Callback to Media Service (if URL is set)
      if (API_CALLBACK_URL) {
        console.log(`Sending callback to ${API_CALLBACK_URL}`);
        try {
          const res = await fetch(API_CALLBACK_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              media_id: mediaId,
              status: 'completed',
              variants: variantsMap
            })
          });
          if (!res.ok) {
            console.error('Callback failed with status:', res.status);
          }
        } catch (fetchErr) {
          console.error('Callback request failed:', fetchErr);
        }
      }

    } catch (error) {
      console.error(`Error processing ${srcKey} from bucket ${srcBucket}:`, error);
      
      if (API_CALLBACK_URL) {
        try {
          await fetch(API_CALLBACK_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              media_id: mediaId,
              status: 'failed',
              error_message: error.message
            })
          });
        } catch (e) {}
      }
    }
  }
};

const streamToBuffer = (stream) =>
  new Promise((resolve, reject) => {
    const chunks = [];
    stream.on("data", (chunk) => chunks.push(chunk));
    stream.on("error", reject);
    stream.on("end", () => resolve(Buffer.concat(chunks)));
  });
