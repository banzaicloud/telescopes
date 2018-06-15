import {Injectable} from '@angular/core';
import {Product} from './product';
import {Observable, of} from 'rxjs';

export const PRODUCTS: Product[] = [
  {type: "m5.large", cpus: 4, mem: 8,},
  {type: "m5.xlarge", cpus: 8, mem: 16},
]

@Injectable({
  providedIn: 'root'
})
export class ProductService {

  constructor() {
  }

  getProducts(): Observable<Product[]> {
    return of(PRODUCTS);
  }
}
