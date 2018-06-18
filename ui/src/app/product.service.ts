import {Injectable} from '@angular/core';
import {DisplayedProduct, Products, Region} from './product';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';
import {HttpClient} from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ProductService {

  private productsUrlBase = 'api/v1/';

  constructor(private http: HttpClient) {
  }

  getRegions(provider): Observable<Region[]> {
    return this.http.get<Region[]>(this.productsUrlBase + "regions/" + provider)
  }


  getProducts(provider, region): Observable<DisplayedProduct[]> {
    return this.http.get<Products>(this.productsUrlBase + "products/" + provider + "/" + region).pipe(
      map(res => {
        return res.products.map(
          res => {
            return new DisplayedProduct(
              res.type,
              res.cpusPerVm + " vCPUs",
              res.memPerVm + " GB",
              "$" + res.onDemandPrice,
              "$")
          })
      })
    )
  }
}
