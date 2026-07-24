FROM oven/bun:1 AS build

WORKDIR /app

ARG VITE_API_URL=http://localhost:8080/api/v1
ENV VITE_API_URL=$VITE_API_URL

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .
RUN ./node_modules/.bin/vite build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist/client /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
