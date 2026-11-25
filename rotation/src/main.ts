import 'reflect-metadata'
import {
  Body, Controller, Headers, HttpCode, HttpStatus, Injectable, Logger, Module,
  ServiceUnavailableException, UnauthorizedException, BadRequestException, Post,
} from '@nestjs/common'
import { NestFactory } from '@nestjs/core'
import { randomInt, timingSafeEqual } from 'node:crypto'
import { setTimeout as delay } from 'node:timers/promises'

const mainURL = process.env.MAIN_API_URL ?? 'http://127.0.0.1:8080'
const key = process.env.ROTATION_KEY ?? 'rotate08'

type Position = { angle_x: number; angle_y: number; rotation_version: number }

@Injectable()
class RotationService {
  private readonly jobs = new Set<number>()
  private readonly log = new Logger(RotationService.name)

  enqueue(id: number) {
    if (this.jobs.has(id)) return
    if (this.jobs.size >= 128) throw new ServiceUnavailableException('Очередь заполнена')
    this.jobs.add(id)
    void this.rotate(id).catch((error: unknown) => this.log.error(`Вид затрат ${id}: ${String(error)}`))
      .finally(() => this.jobs.delete(id))
  }

  private async rotate(id: number) {
    const current = await fetch(`${mainURL}/api/breakeven_point/feed/${id}`, { signal: AbortSignal.timeout(3000) })
    if (!current.ok) throw new Error(`GET: ${current.status}`)
    const { angle_x, angle_y, rotation_version } = await current.json() as Position
    await delay(randomInt(5000, 10001))
    const result = await fetch(`${mainURL}/internal/costs/${id}/rotation`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', 'X-Rotation-Key': key },
      body: JSON.stringify({ angle_x: (angle_x + 45) % 360, angle_y: (angle_y + 75) % 360, rotation_version }),
      signal: AbortSignal.timeout(3000),
    })
    if (result.status === 409) {
      this.log.debug(`Вид затрат ${id}: сохранена более новая позиция`)
    } else if (!result.ok) {
      throw new Error(`PUT: ${result.status}`)
    }
  }
}

@Controller('rotations')
class RotationController {
  constructor(private readonly service: RotationService) {}

  @HttpCode(HttpStatus.ACCEPTED)
  @Post()
  create(@Body() body: { id?: number } | undefined, @Headers('x-rotation-key') token = '') {
    const supplied = Buffer.from(token)
    const expected = Buffer.from(key)
    if (supplied.length !== expected.length || !timingSafeEqual(supplied, expected)) {
      throw new UnauthorizedException()
    }
    const id = body?.id
    if (typeof id !== 'number' || !Number.isSafeInteger(id) || id < 1 || id > 2_147_483_647) {
      throw new BadRequestException('Укажите положительный id вида затрат')
    }
    this.service.enqueue(id)
    return { id, status: 'accepted' }
  }
}

@Module({ controllers: [RotationController], providers: [RotationService] })
class AppModule {}

async function bootstrap() {
  const app = await NestFactory.create(AppModule)
  await app.listen(Number(process.env.PORT ?? 3100), process.env.HOST ?? '127.0.0.1')
}

void bootstrap()
