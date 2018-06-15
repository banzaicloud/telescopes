import {Injectable} from '@angular/core';
import {Product, Products} from './product';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';
import {HttpClient} from '@angular/common/http';

export const PRODUCTS: Product[] = [
  {type: "m5.large", cpusPerVm: 4, memPerVm: 8,},
  {type: "m5.xlarge", cpusPerVm: 8, memPerVm: 16},
]

@Injectable({
  providedIn: 'root'
})
export class ProductService {

  private productsUrl = 'api/v1/products/ec2/eu-west-1';

  constructor(private http: HttpClient) {
  }

  getProducts(): Observable<Product[]> {
    return this.http.get<Products>(this.productsUrl).pipe(
      map(res => {
        return res.products
      })
    )
  }
}
