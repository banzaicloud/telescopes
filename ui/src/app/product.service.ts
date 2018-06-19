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
        return res.Products.map(
          res => {
            var avgSpot = 0;
            if (res.spotPrice != null) {
              var i;
              for (i = 0; i < res.spotPrice.length; i++) {
                avgSpot = avgSpot + parseFloat(res.spotPrice[i].price);
              }
              avgSpot = avgSpot / res.spotPrice.length;
            }
            var displayedSpot = "$" + avgSpot.toFixed(5);
            if (avgSpot == 0) {
              displayedSpot = "unavailable"
            }
            return new DisplayedProduct(
              res.type,
              res.cpusPerVm + " vCPUs",
              res.memPerVm.toFixed(2) + " GB",
              "$" + res.onDemandPrice.toFixed(5),
              displayedSpot,
              res.ntwPerf == "" ? "unavailable" : res.ntwPerf)
          })
      })
    )
  }
}
