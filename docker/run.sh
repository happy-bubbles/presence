docker run -ti -p 5555:5555 \
--restart=unless-stopped \
--log-opt max-size=2m \
--name="happy-bubbles" happy-bubbles/happy-bubbles
