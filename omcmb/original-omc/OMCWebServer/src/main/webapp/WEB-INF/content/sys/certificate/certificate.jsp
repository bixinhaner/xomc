<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
	#certificate{
		height: 100%;
		display: flex;
		width: 100%;
	}
	#certificate .el-tabs {
		width: 100%;
		height: 100%;
	}		
	#certificate .el-textarea__inner {
		line-height: 1.2;
		padding: 2px 15px;
		resize: none;
	}	
	#certificate .el-upload__tip {
		margin-top: 0;
	}
	#certificate .slide-content{
		margin-left:0 !important;
	}
	#certificate .el-dialog__body{
		padding:24px 20px 0; 
		min-height:98px;	
	}
	#certificate .el-dialog__footer{
		padding:20px 20px 26px;
		text-align:left;
	}
	#certificate .el-checkbox-group{
		padding:16px 0 10px;
	}
	#certificate .issueStatus{ 
		margin-top:4px;
		font-size:20px; 
		margin-right:10px;
		float:left;
	}
	#certificate .issueNoColor:before{		
		color:#CFCFCF;
	}
	#certificate .issueOkColor:before{
		color:#67D972;
	}
	#certificate .issueErrorColor:before,
	#certificate .statusError:before{
		color:#E88282;
	}
	#certificate .statusTip{
		overflow:hidden;
		text-overflow:ellipsis;
		white-space:nowrap;
		font-size:12px !important;
	}
	#certificate .el-checkbox__input.is-checked+.el-checkbox__label{
		color:#333;
		font-weight:normal;
	}
	#certificate .caCertificateTitle{
		right: 15px;
		top: 41px;
		cursor: pointer;
	}
	#certificate .el-form-item{
		margin-bottom:26px !important;
	}
	#certificate .el-form-item__content{
		line-height:26px;
	}
	#certificate .el-message-box{
		padding-bottom:20px;
	}
	#certificate .el-message-box__btns > .el-button:not(:first-child){
   		 margin: 0 15px 0 0;
	}
	#certificate .el-message-box__content{
		padding:30px;
	}
	#certificate .el-message-box__message p {
		margin:0 !important;
	}
	#certificate .el-tooltip__popper{
		padding:10px;
	}
	#certificate .statusProgress{
		font-size:12px !important;
	}
	/* 证书自动更新 */
	#certificate .enableText {margin-left: 5px;}
	#certificate .autoUpdateDialog {text-align: right;}
	#certificate .el-tabs__active-bar.is-top{
		bottom: 0 !important;
	}
	#certificate .commonTabsTop .el-tabs__header {
		border-bottom: 1px solid #D5DCEC;
	}
	#certificate .el-input-group__append {
		padding: 0 6px;
	}
	.jumpManagePageSlide .slide-content{
		background: #F6F7FB;
	}
	#certificate .jumpSSLCertIcon:hover::after {
		margin-left: -55px;
	}
    #certificate .sendCertFormWarp .el-form-item .el-form-item__error {
    	margin: 0 0 0 106px;
    }
    #certificate .sendCertFormWarp .el-form-item__label {
    	line-height: 26px;
    }
    [class*="el-icon-status-issued"] {
		cursor: pointer;
	}
	#certificate .certUpdateIcon:before {
		font-size: 18px !important;
		color: rgba(0, 0, 0, 0.8);
	}
	.last-bt-tips:hover::after {
		margin-left: -55px;
	}
</style>
<!--设备证书页面  -->
<div class="overflow-cls">
	<div id="certificate" class='commonFlex' style="min-width: 900px;position: relative;overflow-y: hidden;">
		<div class='leftWarp commonWarp'>	
			<!-- eNB IPSec 证书操作项 -->
			<div v-if="activeName == 'ipsecCertificate'">
				<div class="newIconBoxCls-bt" style="right: 60px; top: 41px;" @click="jumpCertManagePage" tip="<%=rb.getString("OMCZhengShuGuanLi")%>">
					<span class='el-icon el-icon-status-upload'></span>
				</div>
				<div v-if="hasLicenseRole" class="newIconBoxCls-bt" style="right: 20px; top: 41px;" @click="updateSetting" tip="<%=rb.getString("SheZhi")%>">
					<span class='el-icon el-icon-operation-settings'></span>
				</div>
			</div>
			<!-- eNB SSL 证书操作项 -->
			<div v-if="activeName == 'sslCertificate'">
                <!-- 批量输入选中 -->
                <div v-if="hasLicenseRole" class="newIconBoxCls-bt" style="right:128px;top:41px;" @click="batchInputSslCertificate" tip="<%=rb.getString("PiLiangShuRu")%>">
                    <span class='el-icon el-icon-batchInput'></span>
                </div>
				<!-- 全部设备下发证书 -->
				<div v-if="hasLicenseRole" class="newIconBoxCls-bt" style="right: 92px; top: 41px;" @click="allDeviceIssuedCertificate" tip="<%=rb.getString("XiaFaZhengShu")%>">
					<span class='el-icon el-icon-status-issued'></span>
				</div>
				<!-- SSL 证书维护 -->
				<div class="newIconBoxCls-bt" style="right: 56px; top: 41px;" @click="jumpSslCertificatePage" tip="<%=rb.getString("SSLZhengShu")%>">
					<span class='el-icon el-icon-circle-certificate'></span>
				</div>
                <!-- SSL 证书导出 -->
				<div class="newIconBoxCls-bt" style="right: 20px; top: 41px;" @click="exportSslCertificate" tip="<%=rb.getString("DaoChu")%>">
					<span class="el-icon-operation-export el-icon"></span>
				</div>
			</div>
			
			<el-tabs v-model="activeName" @tab-click='tabClick' class='commonTabsTop'>			 	
			 	<!-- 4g Ipsec Certificate :url="ipsecCertUrl"-->
			 	<el-tab-pane label="<%=rb.getString("IpsecZhengShu")%>" name="ipsecCertificate">
	 				<el-ctable ref="ipsecCertCtable" :time="6" :url="ipsecCertUrl" :query-params="ipsecCertQueryParams" id="ipsecCertCtable" :limit="limitBatch"
		            	:page-size="pageSize" pagination="true" :rownumber=true :row-key="'serialNumber'" @selection-change='ipsecCertBatchSelect' @row-click="ipsecCertRowClick">
		                <template slot="toolbar">
		                	<div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
		                     	<!-- 已选数据 -->
		                 		<div v-if="hasLicenseRole" class="selectBlukBoxCls">
		                            <div class="selectMain headBtnItemCls">
		                             	<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable" style='border-right: 0; padding: 0;'>
											<span class="el-icon-selected el-icon"></span>
											<span class="bulkSelectNumBoxCls">( {{certSelectData.length}} )</span>
										</div>
		                                 <div class="selectTableBoxCls" v-show="bulkSelectShow" style="position: absolute;top: 32px;left: 30px; ">
		                                     <div class="selectBoxTitle">
		                                         <span><%=rb.getString("YiXuan")%></span>
		                                         <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
		                                     </div>
		                                     <div class="selectBoxMain">
		                                         <div class="tableInfoCls">
		                                             <div class="tableInfoHeader">
		                                                 <div><%=rb.getString("XiaoZhanBianMa")%></div>
		                                                 <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
		                                             </div>
		                                             <el-ctable 
		                                                 id="bulkSelectTable" 
		                                                 ref="bulkSelectTable" 
		                                                 :data="certSelectData" 
		                                                 :showHeader="false"
		                                                 :rownumber="false"
		                                                 :front-pagination="true"
		                                                 height="270px" pagination="true" >
		                                                 <el-table-column prop="id" v-if="false"></el-table-column>
		                                                 <el-table-column width="588">
		                                                     <template slot-scope="scope" >
		                                                         <div class="tableItemCls">
		                                                             <span>{{scope.row.serialNumber}}</span>
		                                                             <span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
		                                                         </div>
		                                                     </template>
		                                                 </el-table-column>
		                                             </el-ctable>
		                                         </div>
		                                     </div>
		                                 </div>
		                             </div>
		                        </div>
		                        <div v-if="hasLicenseRole" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCertUpdate">
		                     		<span class='el-icon el-icon-operation-CertUpdate'></span>
		                     		<span><%=rb.getString("LiJiGengXinZhengShu")%></span>
		                     	</div>
		                      	<div v-if="hasLicenseRole" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCertAutoUpdateEnable('on')">
		                     		<span class='el-icon el-icon-operation-CertUpdateEnable'></span>
		                     		<span><%=rb.getString("KaiQiZiDongGeng")%></span>
		                     	</div>
		                     	<div v-if="hasLicenseRole" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCertAutoUpdateEnable('off')"  style='border-right: 0;'>
		                     		<span class='el-icon el-icon-operation-CertUpdateDisable'></span>
		                     		<span><%=rb.getString("GuanBiZiDongGeng")%></span>
		                     	</div>
		                     </div>
		                	 <div class='toolbarHeadBtnBoxCls commonQuery tableHeadQueryBoxCls' style=' border: 0; padding: 5px 0;'>
								<el-query type="normal" @query="ipsecCertQuery" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("IpsecZhengShu")%>"></el-query>
		                    	
		                    	<div v-for="(item,index) in advancedQueryItemList" style='margin-left: 10px;'>
		                            <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
										<el-popfilter
											:label='item.label'
											v-model="item.checkedItemList"
											:list="item.options"
											:visible.sync="item.isShow"
											@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
										</el-popfilter>
									</div>
									<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
										<el-popfilter
											type="single"
											:label='item.label'
											v-model="item.selectVal"
											:list="item.options"
											:visible.sync="item.isShow"
											@check-change="advanceQuery(item.type,item.value,item.selectVal)">
										</el-popfilter>
									</div>
		                        </div>
		                        <div class="advancedQueryItemBox"  style="background: #FFF; margin-left: 10px;" @click="clearFilterClick">
		                            <%=rb.getString("QingKongShaiXuan")%>
		                        </div>
							</div>                   
		                </template>              
						<el-table-column v-if="hasLicenseRole" type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column v-if="hasLicenseRole" width="40">
		                    <template slot-scope="scope">
		                   		<div>
									<span class="el-icon el-icon-operation-CertUpdate certUpdateIcon" v-if="scope.row.certAutoEnroll == '1'" @click="singleCertUpdate(scope.row,event)"></span>
									<span class="el-icon el-icon-operation-CertUpdate disabled" v-else-if="scope.row.certAutoEnroll == '0'"></span>
									<span v-else></span>
								</div>
		                    </template>
		                </el-table-column>
		                <el-table-column label="<%=rb.getString("CaoZuoZhuangTai")%>" prop="executeStatus" min-width="160" show-overflow-tooltip>
		                	<!--操作状态 : sendCert-下发证书、backupCert -备份证书、updateCert-更新证书、normal-正常-->	
		                	<template slot-scope="scope">				
								<div v-if="scope.row.executeStatus == 'sendCert'" class="statusProgress">			
									<span class="status_backup" style="float:left;"></span>
									<span><%=rb.getString("PeiZhiZhengShuJinXingZhong")%></span>
								</div>
								<div v-else-if="scope.row.executeStatus == 'backupCert'" class="statusProgress">			
									<span class="status_backup" style="float:left;"></span>
									<span><%=rb.getString("BeiFeiJinXingZhong")%></span>
								</div>
								<div v-else-if="scope.row.executeStatus == 'updateCert'" class="statusProgress">			
									<span class="status_backup" style="float:left;"></span>
									<span><%=rb.getString("ZhengShuGengXinZhong")%></span>
								</div>
								<div v-else-if="scope.row.executeStatus == 'normal'">
									<span><%=rb.getString("ZhengChang")%></span>
								</div>
							</template>
		                </el-table-column> 
		                <el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serialNumber" min-width="150" show-overflow-tooltip></el-table-column>   
		                <el-table-column label="<%=rb.getString("ZhengShuZhuangTai")%>" prop="certStatus" min-width="100" show-overflow-tooltip sortable>
		                	<!-- 证书当前状态 : 0 == 下发失败，1 == 已下发 -->	
		                	<template slot-scope="scope">				
								<div v-if="scope.row.certStatus == '0'" class='commonFlex'>	
									<span class="el-icon el-icon-status-failed-take-effect issueStatus issueErrorColor" style='margin-top:2px;'></span>
									<span><%=rb.getString("WuXiaoDe")%></span>
								</div>
								<div v-if="scope.row.certStatus == '1'">
									<span class="el-icon el-icon-status-take-effect issueStatus issueOkColor" style='margin-top:2px;'></span>
									<span><%=rb.getString("YouXiaoDe")%></span>
								</div>
							</template>
		                </el-table-column>                            
		                <el-table-column label="<%=rb.getString("IpsecZhengShu")%>" prop="certName" min-width="150" show-overflow-tooltip></el-table-column>              
						<el-table-column label="<%=rb.getString("YouXiaoKaiShiShiJian")%>" prop="certValidStartTime" min-width="150" show-overflow-tooltip></el-table-column>
						<el-table-column label="<%=rb.getString("YouXiaoJieShuShiJian")%>" prop="certValidEndTime" min-width="150" show-overflow-tooltip></el-table-column>	
						<el-table-column label="<%=rb.getString("ZhengShuZiDongGengXinKaiGuan")%>" prop="certAutoEnroll" min-width="180" show-overflow-tooltip sortable style="position:relative;">
							<template slot-scope="scope">
								<div v-if="scope.row.certAutoEnroll == '1' || scope.row.certAutoEnroll == '0'" >
									<el-switch :ref="scope.row.serialNumber" v-model="scope.row.certAutoEnroll" style="height: 18px;"  
										:disabled="!hasLicenseRole"
										:before-change="autoEnableChange"
										:active-value="'1'" 
										:inactive-value="'0'" 
										active-color="#4D84FF" 
										inactive-color="#CFCFCF">	
									</el-switch>
									<span v-if="scope.row.certAutoEnroll == '1'" class="enableText">Enable</span>
									<span v-if="scope.row.certAutoEnroll == '0'" class="enableText">Disable</span>							
								</div>
								<div v-else>--</div>
							</template>
						</el-table-column>			    
						<el-table-column label="<%=rb.getString("ShangCiGengXinShiJian")%>" prop="certUpdateTime" min-width="150" show-overflow-tooltip sortable></el-table-column>				
						<el-table-column label="<%=rb.getString("GengXinJieGuo")%>" prop="certUpdateStatus" min-width="150" show-overflow-tooltip sortable>
							<!-- 证书当前状态 : 0 == 下发失败，1 == 已下发，2 == 下发中，3 == 未下发 -->	
		                	<template slot-scope="scope">				
								<div v-if="scope.row.certUpdateStatus == '0'" class='failedText'><%=rb.getString("ShiBai")%></div>
								<div v-if="scope.row.certUpdateStatus == '1'"><%=rb.getString("ChengGong")%> </div>
							</template>
		                </el-table-column>	
						<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="certUpdateFailureReason" min-width="180" show-overflow-tooltip></el-table-column>
		            </el-ctable>           
			 	</el-tab-pane>
			 
			 	<!-- 4g-SSL 证书   -->
			 	<el-tab-pane label="<%=rb.getString("TRSSLZhengShu")%>" name="sslCertificate">
	 				<el-ctable ref="sslCertTable" :time="6" :url="sslCertificateUrl" :query-params="sslCertificateParams" id="sslCertTable" :limit="limitBatch"
		            	:page-size="pageSize" pagination="true" :rownumber=true :row-key="'serialNumber'" @selection-change='sslCertBatchSelect'>
		                <template slot="toolbar">
		                <div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
		                      	<!-- 已选数据 -->
		                 		<div v-if="hasLicenseRole" class="selectBlukBoxCls">
		                             <div class="selectMain headBtnItemCls"  style='border-right: 0; padding: 0;'>
			                             <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
											<span class="el-icon-selected el-icon"></span> 
											<span class="bulkSelectNumBoxCls">( {{certSelectData.length}} )</span>
										</div>
										<div class="selectTableBoxCls" v-show="bulkSelectShow" style="position: absolute;top: 32px;left: 30px;">
		                                     <div class="selectBoxTitle">
		                                         <span><%=rb.getString("YiXuan")%></span>
		                                         <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
		                                     </div>
		                                     <div class="selectBoxMain">
		                                         <div class="tableInfoCls">
		                                             <div class="tableInfoHeader">
		                                                 <div><%=rb.getString("XiaoZhanBianMa")%></div>
		                                                 <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
		                                             </div>
		                                             <el-ctable 
		                                                 id="bulkSelectTable" 
		                                                 ref="bulkSelectTable" 
		                                                 :data="certSelectData" 
		                                                 :showHeader="false"
		                                                 :rownumber="false"
		                                                 :front-pagination="true"
		                                                 height="270px" pagination="true" >
		                                                 <el-table-column prop="id" v-if="false"></el-table-column>
		                                                 <el-table-column width="588">
		                                                     <template slot-scope="scope" >
		                                                         <div class="tableItemCls">
		                                                             <span>{{scope.row.serialNumber}}</span>
		                                                             <span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
		                                                         </div>
		                                                     </template>
		                                                 </el-table-column>
		                                             </el-ctable>
		                                         </div>
		                                     </div>
		                                 </div>
		                             </div>
		                        </div>
		                      	<div v-if="hasLicenseRole" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="singleDeviceIssuedCertificate" style='border-right: 0;'>
		                     		<span class='el-icon el-icon-status-issued'></span>
		                     		<span><%=rb.getString("FaSongZhengShu")%></span>
		                     	</div> 
		                     </div>
		                     <div class='commonQuery' style='display: flex;align-items: center;height:45px;'>
		                     	<el-query type="normal" @query="querySSLCertificate" placeholder='<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("SSLZhengShu")%>' style="margin-right: 10px;"></el-query>
                                 <el-popfilter
                                    style="margin-right: 10px;"
                                    label='<%=rb.getString("ZaiXianZhuangTai") %>'
                                    v-model="sslCertificateParams.connection_status"
                                    type="single"
                                    :list="connectionStatusList">
                                </el-popfilter>
                                <el-popfilter
                                    style="margin-right: 10px;"
                                    label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'
                                    v-model="sslCertificateParams.productType"
                                    type="single"
                                    :list="headerProductList.map(item=>{return {label:item.name,value:item.value}})">
                                </el-popfilter>
                                <el-popfilter
                                    style="margin-right: 10px;"
                                    label='<%=rb.getString("ZhengShuJianCe")%>'
                                    v-model="sslCertificateParams.certVerifyStatus"
                                    type="single"
                                    :list="certVerifyStatusList">
                                </el-popfilter>
                                <div class="pop-filter-clear" style="margin: 0 5px;" 
                                    @click="resetQuery">
                                    <%=rb.getString("QingKongShaiXuan")%>
                                </div>
                            </div>
		                </template>
		                <el-table-column v-if="hasLicenseRole" type="selection" :reserve-selection="true"></el-table-column>
                        <el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div v-html="connStatusFormatter(scope.row.connection_status,scope.row)"></div>
							</template>
                        </el-table-column>
		                <el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber" show-overflow-tooltip min-width="160"></el-table-column>   
                        <el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' prop="productType" min-width="120"></el-table-column>
		                <el-table-column label='<%=rb.getString("SSLZhengShu")%>' prop="sslCertName" show-overflow-tooltip  min-width="150">
		                	<!-- CA证书当前状态 : 0 == 失败，1 == 成功，2 == 进行中，3 == 默认值未执行 -->	
		                	<template slot-scope="scope">				
								<div v-if="scope.row.status == '0'" class="statusTip">	
									<el-tooltip content="<%=rb.getString("ZhengShuXiaFaShiBai")%>" placement="bottom">			
										<span class="el-icon el-icon-status-issued issueStatus issueErrorColor"></span>
									</el-tooltip>
									{{scope.row.sslCertName}}
								</div>
								<div v-if="scope.row.status == '1'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuYiXiaFa")%>" placement="bottom">				
										<span class="el-icon el-icon-status-issued issueStatus issueOkColor"></span>
									</el-tooltip>
									{{scope.row.sslCertName}}
								</div>
								<div v-if="scope.row.status == '2'">						
									<div class='commonFlex'>
										<el-tooltip content="<%=rb.getString("ZhengShuXiaFaZhong")%>"  placement="bottom">						
											<span class="status_issuing" style='margin-top: 5px; margin-right: 10px;'></span>
										</el-tooltip>
										<span class="statusProgress">{{scope.row.sslCertName}}</span>
									</div>
								</div>
								<div v-if="scope.row.status == '3' " class="statusTip">				
									<el-tooltip content="<%=rb.getString("ZhengShuWeiXiaFa")%>" placement="bottom">				
										<span v-if="scope.row.sslCertName !=null &&  scope.row.sslCertName !=''" class="el-icon el-icon-status-issued issueStatus issueNoColor" ></span> 
									</el-tooltip>
									{{scope.row.sslCertName}}
								</div>
							</template>
		                </el-table-column>   
                        <el-table-column label='<%=rb.getString("ZhengShuJianCe")%>' prop="certVerifyStatus" show-overflow-tooltip min-width="130">
                            <template slot-scope="scope">
                                <span v-if="scope.row.certVerifyStatus == '1'" style="color: #0ABF5B;"><%=rb.getString("ZhengShuKeYong")%></span>
                                <span v-if="scope.row.certVerifyStatus == '0'"><%=rb.getString("ZhengShuBuKeYong")%></span>
                            </template>
                        </el-table-column>
		                <el-table-column label='<%=rb.getString("YouXiaoKaiShiShiJian")%>' prop="validStartTime" show-overflow-tooltip min-width="150"></el-table-column>
						<el-table-column label='<%=rb.getString("YouXiaoJieShuShiJian")%>' prop="validEndTime" show-overflow-tooltip min-width="150"></el-table-column>	
                        <el-table-column label='<%=rb.getString("FaSongShiJian")%>' prop="updateTime" show-overflow-tooltip min-width="150"></el-table-column>
                        <el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason" show-overflow-tooltip min-width="140"></el-table-column>
		            </el-ctable>           
				</el-tab-pane>
			</el-tabs>
		</div>
	
		<!--IPsec 证书  自动更新证书 配置页面 -->
		<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='showSettingDialog'>
			<div class='rightWarpLayer'>
		 		<div class='rightBoxHeaderHasTip'>
					<div class='headerText'>
						<span><%=rb.getString("SheZhi")%></span>
						<span class='closeIconBox' @click='clearSettingDialog'><i class='el-icon el-icon-close'></i></span>
					</div>
				</div>
				<div class='rightWarpLayerContent'>
					<el-form :model="settingForm" ref="settingForm" :rules='settingRules' label-position="top" style='padding: 30px 20px;'>   
						<el-form-item label="<%=rb.getString("IPDiZhi")%>" prop="ipAddress">
							<el-input v-model="settingForm.ipAddress" style="width: 320px;"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("GengXinZhouQi")%>" prop="updatePeriod" style='margin-bottom: 2px !important;'>
							<el-input  v-model="settingForm.updatePeriod" style="width: 320px;">
								<template slot="append"><%=rb.getString("Tian")%></template>
							</el-input>
						</el-form-item>
						<span v-show="updatePeriodShow" class="commonNotes12"><%=rb.getString("FanWei")%>:1~365</span>
					</el-form>
				</div>
				<div class='commonFlex commonBorderTop commonFormFotter'>
					<div>
						<el-button type="primary" @click="settingSubmit"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="clearSettingDialog"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</div>
			</div>
		</div>
		
		<!--IPsec 证书 自动更新开关打开 弹窗-->
		<el-dialog title="<%=rb.getString("QueRen")%>" :visible.sync="certAutoUpdateDialog" width="500" 
			:close-on-click-modal="false" top="30vh" @close="certAutoUpdateDialog = false">
			<div style="margin-bottom:10px;"><%=rb.getString("QueDingDaKaiZiDongGengXin")%></div>
			<div style="font-size:12px;color:#B4B4B4"><%=rb.getString("IPDiZhi")%>: {{currentIpAddress}}； <%=rb.getString("GengXinZhouQi")%>: {{currentUpdatePeriod}} <%=rb.getString("Tian")%></div>
			<span slot="footer" class="dialog-footer">
				<div class="buttonGroup autoUpdateDialog">
					<el-button type="primary" @click="certAutoUpdateSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="certAutoUpdateDialog = false"><%=rb.getString("QuXiao")%></el-button>
				</div>	
			</span>
		</el-dialog>
				
		<!--SSL 证书 给全部设备下发证书-->
		<el-dialog title='<%=rb.getString("FaSongZhengShu")%>' :visible.sync="showSendSSLCertificate" width="500" class="sendCertFormWarp"
			:close-on-click-modal="false" @close="cancelSendCertificate">
			<el-form :model="sendSSLCertForm" ref="sendSSLCertForm" :rules="sendSSLCertRule" label-position="left" label-width="110">
				<span style="display:block; margin-bottom: 20px;"><%=rb.getString("ShiFouXiaFaZhengShuDaoOMC")%></span>
                <el-form-item v-show="sendSSLCertFlag == 'all'" label='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' prop="productType" placeholder='<%=rb.getString("QingXuanZe") %>'>
	                <el-select v-model="sendSSLCertForm.productType" @change="sendSSLCertProductTypeChange">
	                    <el-option v-for="item in rightProductList" :label="item.name" :value="item.value"></el-option>
	                </el-select>
	            </el-form-item>
				<el-form-item label='<%=rb.getString("SSLZhengShu") %>' prop="sslCert" placeholder='<%=rb.getString("QingXuanZe") %>'>
	                <el-select v-model="sendSSLCertForm.sslCert">
	                    <el-option v-for="item in optionalSslCertList" :label="item.text" :value="item.value"></el-option>
	                </el-select>
	            </el-form-item>
			</el-form>		
			<div slot="footer">
				<div class="buttonGroup">
					<el-button type="primary" @click="confirmSendCertificate"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="cancelSendCertificate"><%=rb.getString("QuXiao")%></el-button>
				</div>	
			</div>
		</el-dialog>

         <!-- 批量选中 -->
         <el-dialog class='dialogStyle' title='<%=rb.getString("PiLiangShuRu")%>' width='630px' :visible.sync='batchInputDialogShow' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
            <el-form ref='batchInputForm' :rules='batchInputRules' :model='batchInputForm' label-position="top">
                <div>
                    <label><%=rb.getString("XiaoZhanBianMa")%></label>
                    <el-form-item prop='serialNumber' style="margin-bottom:22px;">
                        <el-input v-model='batchInputForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
                    </el-form-item>
                    <p style='display:flex;color:#BBB'><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
                </div>
                <div style='margin-top:45px;'>
                    <el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
                    <el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
                </div>
            </el-form>
        </el-dialog>
		
		<el-slide ref="slideCertManagePage" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition" class='jumpManagePageSlide'
			:width="slideWidth" @cancel='cancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>	 
	</div>
</div>
<script type="text/javascript">
	var certificateVue = new Vue({
	    el: '#certificate',
	    data() {
			var vm = this,
			
			validatorIpAddress = (rule,value,callback) => {				
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("IPDiZhiBuNengWeiKong")%>'))
				}else{
					callback();
				}
			},
			validateUpdatePeriod = (rule,value,callback) => {
				var numReg = /^\s*\d+\s*$/;
				if(value === '' || value === null || value === undefined){
					this.updatePeriodShow = false;
					callback(new Error('<%=rb.getString("FanWei")%>:1~365'))
				}else{
					if(numReg.test(value) && (value - 1 >= 0) && (value - 365 <= 0)){
						this.updatePeriodShow = true;
						callback();
					}else{
						this.updatePeriodShow = false;
						callback(new Error('<%=rb.getString("FanWei")%>:1~365'))
					}
				}
			},
			validateSSLCert = (rule,value,callback) => {				
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeSSLZhengShu")%>'))
				}else{
					callback();
				}
			},
            validatorNum = (rule,value,callback) => {
                var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
                    list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
						return item.length > 0;
					});
        
                if (serialNumber == null || serialNumber.length == 0) {
                    callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
                }else {
                    var nameFlag = list.every(function(item,index){
                        return temp.test(item)
                    })
                    if(nameFlag){
                        callback()
                    }else{
                        callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
                    }
                }
            };
	    	return {
	    		activeName: 'ipsecCertificate',
	    		pageSize: 50,
                connectionStatusList:[
                    {label:'<%=rb.getString("QuanBu")%>',value:''},
                    {label:'<%=rb.getString("LianJieZhengChang")%>',value:"1"},
                    {label:'<%=rb.getString("LianJieDuanKai")%>',value:"0"},
                    {label:'<%=rb.getString("TongBuZhong")%>',value:"3"},
                    {label:'<%=rb.getString("TongBuShiBai")%>',value:"2"},
					{label:'<%=rb.getString("ChuShiHuaZhong")%>',value:"4"},
					{label:'<%=rb.getString("YuanTongBuZhong")%>',value:"5"},
					{label:'<%=rb.getString("YuanTongBuWanCheng")%>',value:"6"}
                ],
                certVerifyStatusList:[
                    {label:'<%=rb.getString("QuanBu")%>',value:''},
                    {label:'<%=rb.getString("ZhengShuKeYong")%>',value:"1"},
                    {label:'<%=rb.getString("ZhengShuBuKeYong")%>',value:"0"},
                ],
                headerProductList:[],
                rightProductList:[],
	    		// IPsec Certificate
	    		ipsecCertUrl: "${ctx}/cell/ipsec/cert/queryCellCertList.action",
	            ipsecCertQueryParams: {
	    			timeZone: timeZone,
	             	searchText: '',
					serialNumber :'', 
					certName: '',
					certStatus: '',
					updateStatus: '',
					nbType: 'eNB'
	            },	            
	            certSelectData: [],
				//IPsec Certificate setting
	            showSettingDialog: false,
	            updatePeriodShow: true,
				certAutoUpdateDialog: false,
				enableSingleOrBatch: 'single',
				currentIpAddress: '',
				currentUpdatePeriod: '90',
				updateCertClickRow: [],
				settingForm: {
					ipAddress: '',
					updatePeriod: '90',
				},
				settingRules: {
					ipAddress: [
						{validator: validatorIpAddress, trigger: 'blur'}
					],
					updatePeriod: [
						{validator: validateUpdatePeriod, trigger: 'blur'}
					]
				},
				//SSL Certificate
				sslCertificateUrl: '${ctx}/cert/ssl/getEnbSSLCertInfoPageList.action',
				sslCertificateParams: {
	    			timeZone: timeZone,
	             	searchText: '',
                    connection_status: '',
                    productType: '',
                    certVerifyStatus: ''
	            },
	            
				// send certificate
				showSendSSLCertificate: false,
				sendSSLCertForm: {
                    productType: '',
					sslCert: ''
				},
				sslCertList: [],
				sendSSLCertRule: {
					sslCert: [{validator: validateSSLCert}],
				},
				sendSSLCertFlag: 'all',
				
	            //slider
				slideUrl: '',
				slideTitle: '',
				slideHeader: false,
				slideFooter: false,
				slidePosition: 'top',
				slideModal: false,
				slideWidth: '',
				slideHeight: '',				
	            
				//已选
				bulkSelectShow: false,
				advancedQueryItemList: [
					{
						type: 'select',
						isShow: true,
						popoverShow: false,
						selectVal: '',
						label: '<%=rb.getString("ZhengShuZhuangTai") %>',
						options: [
							{value: '', label: '<%=rb.getString("QuanBu")%>'},
							{value: '0', label: '<%=rb.getString("WuXiaoDe")%>'},
							{value: '1', label: '<%=rb.getString("YouXiaoDe")%>'}
						],
						value: 'certStatus',
					},
					{
						type: 'select',
						isShow: true,
						popoverShow: false,
						selectVal: '',
						label: '<%=rb.getString("GengXinJieGuo") %>',
						options: [
							{value: '', label: '<%=rb.getString("QuanBu")%>'},					
							{value: '0', label: '<%=rb.getString("ShiBai")%>'},
							{value: '1', label: '<%=rb.getString("ChengGong")%>'}
						],
						value: 'updateStatus',
					}
				],
                batchInputDialogShow: false,
                batchInputForm:{
                    serialNumber:'',
                },
                batchInputRules:{
                    serialNumber:[
                        {validator:validatorNum,trigger:'change'}
                    ]
                },
	    	}
	    },
	    methods: {	
	    	//------------------------------------------------------------ tab ---------------------------------------------------------- 
	    	init(){
	    		var vm = this;
	    		vm.getProductList();
	    	},
            // 获取产品类型下拉数据
            getProductList(){
                var vm = this;
                axios.post("${ctx}/cert/ssl/getProductTypeList.action").then(function(res){
                    var data = res.data ? res.data : [];
                    let noAllList = [];
                    let allList = [{name: '<%=rb.getString("QuanBu")%>',value:''}];
                    data.map((item)=>{
                        noAllList.push({name: item,value: item});
                        allList.push({name: item,value: item});
                    })
                    vm.rightProductList = noAllList;
                    vm.headerProductList = allList;		
                }); 
            },
			// License 连接状态格式化
			connStatusFormatter(value,rowData,rowIndex){
				if("initializing" == value) {
					return "<div class='el-icon el-loading status-init'></div>";
				}else if("syncSourceInSync" == value) {
					return "<div class='el-icon el-loading status-sync'></div>";
				}else if("syncSourceInSynced" == value) {
					return "<div class='el-icon el-loading status-synced'></div>";
				}else if ("On" == value) {
					return "<div class='el-icon el-icon-status-conn-on connauto' style='font-size:22px;'></div>";
				} else if("updating" == value){
					return "<div class='status_syncing connauto'></div>";
				}else if ("Exception" == value) {
					return "<div class='conn_exc connauto'></div>";
				} else {
					return "<div class='el-icon el-icon-status-conn-off connauto' style='font-size:22px;'></div>";
				}
				return value;
			},
	    	//tab 切换                                                                     
	    	tabClick(tab){
	 	    	var vm = this;		
	 	    	
				vm.$refs['ipsecCertCtable'].clearSelection();
				vm.$refs["sslCertTable"].clearSelection();
				vm.bulkSelectShow = false;
				vm.showSettingDialog = false;
                if(vm.activeName == 'sslCertificate'){
                    vm.commonSSLCertInfoList();
                }
	 	    },

	    	//----------------------------------------------------------IPsec 证书 Tab ---------------------------------------------------------- 
	 		// 表格选中数据
			ipsecCertBatchSelect(selection){
				var vm = this; 

				vm.certSelectData = selection;
			},			
			// 当前行点击
	    	ipsecCertRowClick(row,column,event){
				var vm = this;
				
				vm.updateCertClickRow = row;
			},	    				
	    	// 单个手动更新
			singleCertUpdate(row){
				var vm = this,
					params = {
						smallCellCodes: row.smallCellCode,
						nbType: 'eNB'
					};

				vm.$confirm('<%=rb.getString("QueDingGengXinZhengShu")%>', '<%=rb.getString("QueRen")%>',{
					customClass: 'warningConfirm',
					confirmButtonText: '<%=rb.getString("QueDing")%>',
					cancelButtonText: '<%=rb.getString("QuXiao")%>',
					closeOnClickModal: false
				}).then(() => {
					axios.post('${ctx}/cell/ipsec/cert/update.action', stringify(params)).then(function(response){
						var data = response.data;
						
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type: 'success',
							});
							vm.$refs.ipsecCertCtable.refresh();
						}else{
							vm.$message({
								message: data.message,
								type: 'error',
							});
						}
						vm.certSelectData = [];
						vm.$refs.ipsecCertCtable.clearSelection();//清空勾选的数据
					}).catch(function(error){})
				}).catch()
	    	},
	    	// 自动更新开关点击事件	    
			autoEnableChange(fn){
	    		var vm = this;
	    		
				vm.$nextTick(function(){			
					if(vm.updateCertClickRow.certAutoEnroll == '0'){
						var params = {	
							smallCellCodes: vm.updateCertClickRow.smallCellCode,
							enableAutoEnroll: '1',
							nbType: 'eNB'
						}
					}else{
						var params = {
							smallCellCodes: vm.updateCertClickRow.smallCellCode,
							enableAutoEnroll: '0',
							nbType: 'eNB'
						}
					}
					//开关为开时
					if(vm.updateCertClickRow.certAutoEnroll == '1'){
						vm.$confirm('<%=rb.getString("QueDingGuanBiZiDongGengXin")%>', '<%=rb.getString("QueRen")%>',{
							closeOnClickModal: false,
							customClass: 'warningConfirm',
							callback: function(valid){
								if(valid == 'confirm') {
									axios.post('${ctx}/cell/ipsec/cert/autoEnroll.action', stringify(params)).then(function(response){
										var data = response.data;
										
										if(data) {
											if(data["success"]){
												fn();
												vm.$message({
													message: '<%=rb.getString("ChengGong")%>',
													type: 'success'
												});
												vm.$refs.ipsecCertCtable.refresh();
											}else{
												vm.$message.error(data["message"])
											}
											
											vm.certSelectData = [];
											vm.$refs.ipsecCertCtable.clearSelection();//清空勾选的数据
										}
									}).catch(function(error){})
								}else{
									vm.updateCertClickRow.certAutoEnroll = '1';
								}
							}
						});
					}else{
						var params = {
								nbType: 'eNB'
							}
						axios.post("${ctx}/cell/ipsec/cert/getConfigInfo.action", stringify(params)).then(function(response){
							var data = response.data;
							
							if(data.ipAddress && data.updatePeriod){
								vm.currentIpAddress = data.ipAddress;
								vm.currentUpdatePeriod = data.updatePeriod;
								vm.certAutoUpdateDialog = true;
								vm.enableSingleOrBatch = 'single';
							}else{
								vm.$message({
									message: '<%=rb.getString("QingxianPeiZhiIpDiZhiTiShi")%>',
									type: 'warning',
								});
							}
						}).catch(function(error){
							
						}) 
					} 
					event.stopPropagation();
				})
	    	},
	    	
	    	// 单个、 批量自动更新开关  保存
			certAutoUpdateSubmit(){
	    		var vm = this, smallCodeList = [], params = {};
	    		
				if(vm.enableSingleOrBatch == 'single'){
					params = {	
						smallCellCodes: vm.updateCertClickRow.smallCellCode,
						enableAutoEnroll: '1',
						nbType: 'eNB'
					};
				}else{
					smallCodeList = vm.certSelectData.map(function(item){ return item.smallCellCode});
					params.smallCellCodes = smallCodeList.join(','); 
					params.enableAutoEnroll = '1';
					params.nbType = 'eNB';
				}
	    		axios.post('${ctx}/cell/ipsec/cert/autoEnroll.action',stringify(params)).then(function(response){
					var data = response.data;
					
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("MingLingYiXiaFa")%>',
							type: 'success',
						});
						vm.certAutoUpdateDialog = false;
						vm.$refs.ipsecCertCtable.refresh();
					}else{
						vm.$message({
							message: data.message,
							type: 'error',
						});
					}
					vm.certSelectData = [];
					vm.$refs.ipsecCertCtable.clearSelection();
					
				}).catch(function(error){
					
				})				
			},
			// 批量操作： 开启或关闭证书自动更新
			batchCertAutoUpdateEnable(status){
	    		var vm = this, smallCodeList = [], params = {};
	    		
	    		if(vm.certSelectData.length == 0) return;
	    		
				if(status === 'on'){
					params.nbType = 'eNB';
					axios.post("${ctx}/cell/ipsec/cert/getConfigInfo.action", stringify(params)).then(function(response){
						var data = response.data;
						
						if(data){
							vm.currentIpAddress = data.ipAddress;
							vm.currentUpdatePeriod = data.updatePeriod;  
							vm.enableSingleOrBatch = 'bacth';
							
							if(data.ipAddress && data.updatePeriod){
								vm.certAutoUpdateDialog = true;
							}else{
								vm.$message({
									message: '<%=rb.getString("QingxianPeiZhiIpDiZhiTiShi")%>',
									type: 'warning',
								});
							}
						}
					}).catch(function(error){})
				}else{				
					smallCodeList = vm.certSelectData.map(function(item){ return item.smallCellCode});
					
					params.smallCellCodes = smallCodeList.join(','); 
					params.enableAutoEnroll = '0';
					params.nbType = 'eNB';
					vm.$confirm('<%=rb.getString("QueDingGuanBiZiDongGengXin")%>', '<%=rb.getString("QueRen")%>',{
						closeOnClickModal: false,
						customClass: 'warningConfirm',
						callback: function(valid){
							if(valid == 'confirm') {
								axios.post('${ctx}/cell/ipsec/cert/autoEnroll.action', stringify(params)).then(function(response){
									var data = response.data;
									
									if(data["success"]){
										vm.$message({
											message: '<%=rb.getString("ChengGong")%>',
											type: 'success'
										});
										
										vm.$refs.ipsecCertCtable.refresh();
										
									}else{
										vm.$message.error(data["message"])
									}
									
									vm.certSelectData = [];
									vm.$refs.ipsecCertCtable.clearSelection();//清空勾选的数据
								}).catch(function(error){})
							}
						}
					});
				}
	    	},
	    	// 批量操作： 立即更新证书
	    	batchCertUpdate(){
	    		var vm = this, smallCodeList = [], params = {};
	    		
				if(vm.certSelectData.length == 0) return;
				
				smallCodeList = vm.certSelectData.map(function(item){ return item.smallCellCode});
				params.smallCellCodes = smallCodeList.join(',');  
				params.nbType = 'eNB';
				
				vm.$confirm('<%=rb.getString("QueDingGengXinZhengShu")%>', '<%=rb.getString("QueRen")%>',{
					customClass: 'warningConfirm',
					confirmButtonText: '<%=rb.getString("QueDing")%>',
					cancelButtonText: '<%=rb.getString("QuXiao")%>',
					closeOnClickModal: false
				}).then(() => {
					axios.post('${ctx}/cell/ipsec/cert/update.action', stringify(params)).then(function(response){
						var data = response.data;
						
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("MingLingYiXiaFa")%>',
								type: 'success',
							});
							vm.$refs.ipsecCertCtable.refresh();
						}else{
							vm.$message({
								message: data.message,
								type: 'error',
							});
						}
						vm.certSelectData = [];
						vm.$refs.ipsecCertCtable.clearSelection();//清空勾选的数据
					}).catch(function(error){
						
					})
				}).catch()
	    	},
	    	// 证书自动更新： 更新设置按钮点击
			updateSetting(){
	    		var vm = this,
	    			params = {							
						nbType: 'eNB'
					};
	    		
	    		axios.post("${ctx}/cell/ipsec/cert/getConfigInfo.action", stringify(params)).then(function(response){
					var data = response.data;
					
					if(data){
						vm.settingForm.ipAddress = data.ipAddress;
						vm.settingForm.updatePeriod = data.updatePeriod;
						vm.showSettingDialog = true;
					}
				}).catch(function(error){ })  
			},
	    	// 更新设置提交
			settingSubmit(){
				var vm = this,
					params = {	
						ipAddress: vm.settingForm.ipAddress,
						updatePeriod: vm.settingForm.updatePeriod,
						nbType: 'eNB'
					};
				
				vm.$refs.settingForm.validate((valid) => {
                   if (valid) {
                	   axios.post('${ctx}/cell/ipsec/cert/updateConfigInfo.action',stringify(params)).then(function(response){
							var data = response.data;
							
							if(data) {
								if(data["success"]){									
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type: 'success'
									});
									vm.$refs.ipsecCertCtable.refresh();
									vm.showSettingDialog = false;
								}else{
									vm.$message.error(data["message"])
								}
							}
						}).catch(function(error){})                    
                   }
               })
			},
			// 更新设置弹窗关闭
			clearSettingDialog(){
				var vm = this;
				
	    		vm.showSettingDialog = false;
	    		vm.$refs.settingForm.resetFields();
			},
	    	//模糊查询
	    	ipsecCertQuery(val){
				var vm = this;
				
				vm.ipsecCertQueryParams.searchText = val;
			},
			//右上角跳转到证书管理
			jumpCertManagePage(){
				var vm = this;
				
				vm.slideHeader = false
	    	   	vm.slideUrl = "${ctx}/cell/cert/omcCertificateManage.action"
	    	    vm.slideFooter = false
	    	    vm.slidePosition = 'top'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
	    	    vm.$refs.slideCertManagePage.showSlide(function(){ });
				vm.showSettingDialog = false;
			},						          
	    	
			//-------------------------------------------------------------SSL 证书 -------------------------------------------------------------
	    	/**
			*  列表选中
			* @param selection{Array}   选中数据
			*/
			sslCertBatchSelect(selection){
				var vm = this; 
				
			    vm.certSelectData = selection.map((item)=>{
					return Object.assign(item,{snNumber: item.serialNumber})
				}); 
			},
            // 打开 批量输入 弹窗
            batchInputSslCertificate(){
                var vm = this;
                vm.batchInputDialogShow = true;
            },
            // 批量输入 SN 保存
            saveBatchSn(){
                var vm = this, 
                    saveBatchUrl = '${ctx}/cert/ssl/getExistsSNList.action', 
                    params = {
                        deviceType: 'eNB',
                        serialNumbers: '',
                    };
                let snStr = vm.batchInputForm.serialNumber;
                let snList = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;})
                params.serialNumbers = snList.join(";");
                
                vm.$refs.batchInputForm.validate((valid) => {
                    if(valid){
                        axios.post(saveBatchUrl, stringify(params)).then((res)=>{
                            var data = res.data;
                            if(data && data.length > 0){		
                                vm.$refs.sslCertTable.appendCheckedRows(data);
                                vm.closeBatchSn();
                            }else{
                                vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
                            }
                        })
                    }
                })
            },
            closeBatchSn(){
                var vm = this;
                vm.batchInputDialogShow = false;
                vm.$refs.batchInputForm.resetFields();
            },
			// 给全部设备下发证书
	    	allDeviceIssuedCertificate(){
	    		var vm = this;
	    		
	    		vm.sendSSLCertFlag = 'all';
	    		
	    		
	    		vm.showSendSSLCertificate = true;
	    	},
	    	//单个或批量下发证书
	    	singleDeviceIssuedCertificate(){
				var vm = this;
				
				vm.sendSSLCertFlag = 'single';				
				if(vm.certSelectData.length == 0) return;
                let isExistDifProduct = new Set(vm.certSelectData.map(item => item.productType)).size > 1;
                if(isExistDifProduct){
                    vm.$message({
                        message: '<%=rb.getString("PiLiangXiaFaSSLZhengShuTiShi")%>',
                        type: 'error'
                    });
                    return;
                }
                var selectProductType = vm.certSelectData[0].productType;
	    		let isExistCert = vm.sslCertList.some(item => item.productType.includes(selectProductType));
                if(!isExistCert){
                    vm.$message({
                        message: '<%=rb.getString("MeiYouKePiPeiZhengShu")%>',
                        type: 'error'
                    });
                    return
                }
                vm.sendSSLCertForm.productType = selectProductType;
                vm.showSendSSLCertificate = true;
	    		
	    	},
            // 判断 所选数据是否存在不同产品类型的数据
            hasDuplicateField(array, field) {
                const flag = new Set(array.map(item => item.field)).size !== 1;
                return flag;
            },
	    	//获取 SSL 证书下拉菜单
	    	commonSSLCertInfoList(){
	    		var vm = this;
	    		
	    		axios.post("${ctx}/cert/ssl/getSelectSSLCertInfoList.action").then(function(response){
					var data = response.data;
					
					if(data && data.length > 0){
						vm.sslCertList = data;
					}
				})
	    	},
            // SSL 证书 产品类型变化
            sendSSLCertProductTypeChange(){
                var vm = this;
                vm.sendSSLCertForm.sslCert = '';
            },
	    	//确认下发 SSL 证书
	    	confirmSendCertificate(){
	    		var vm = this, params = {}, snList = vm.certSelectData.map(function(item){ return item.serialNumber});
	    		
	    		if(vm.sendSSLCertFlag == 'single'){	    			
	    			params.serialNumbers = snList.join(',');
	    			params.fileId = vm.sendSSLCertForm.sslCert;
	    			params.executeType = 'SELECT';
	    		}else{
                                params.productType = vm.sendSSLCertForm.productType;
	    			params.fileId = vm.sendSSLCertForm.sslCert;
	    			params.executeType = 'ALL';
	    		}
	    		
				vm.$refs.sendSSLCertForm.validate((valid) => {
                   if (valid) {
                	   axios.post('${ctx}/cert/ssl/sendSSLCert.action',stringify(params)).then(function(response){
							var data = response.data;
							
							if(data) {
								if(data["success"]){									
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type: 'success'
									});
									vm.$refs.sslCertTable.refresh();
									vm.showSendSSLCertificate = false;
								}else{
									vm.$message.error(data["message"])
								}
								//置空已选
								vm.$refs["sslCertTable"].clearSelection();
							}
						}).catch(function(error){})                    
                   }
               })
	    	},
	    	//取消下发 SSL 证书
	    	cancelSendCertificate(){
				var vm = this;
	    		
				vm.$refs.sendSSLCertForm.resetFields();
	    		vm.showSendSSLCertificate = false;
	    	},
	    	//跳转到 SSL 证书维护页面
	    	jumpSslCertificatePage(){
				var vm = this;
				
				vm.slideHeader = false
	    	   	vm.slideUrl = "${ctx}/cert/ssl/toSSLCertFileManage.action"
	    	    vm.slideFooter = false
	    	    vm.slidePosition = 'top'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
	    	    vm.$refs.slideCertManagePage.showSlide(function(){});
	    	},
	    	//ssl 证书 查询
	    	querySSLCertificate(val){
	    		var vm = this;
	    		
	    		Object.assign(vm.sslCertificateParams,{	    			
	    			searchText: val	    		
	    		}) 
	    	},
            // 查询重置
            resetQuery(){
                var vm = this,
                    params = {
                        connection_status: '',
                        productType: '',
                        certVerifyStatus: ''
                    };
                Object.assign(vm.sslCertificateParams,params);
            },
	    	//关闭一级slide，跳转到 omc 证书管理页面或者跳转到ssl 证书维护页面
			cancelSlide(){	    		
	    		var vm = this;
	    		
	    		vm.$refs.slideCertManagePage.hide();
                vm.commonSSLCertInfoList();
	    	},
            // 导出SSL证书
            exportSslCertificate(){
                var vm = this,
				exportUrl ='${ctx}/cert/ssl/exportEnbSSLCertInfo.action';
                exportByForm(exportUrl,vm.sslCertificateParams);
            },
			//------------------------------------------------------------ 下拉菜单查询 ------------------------------------------------------------
			// 高级查询 确定事件
			advanceQuery(type,paramsItem,value){
				var vm = this,
					params ={};
				if(type == 'select'){
					params[paramsItem] = value;
				}else{
					params[paramsItem] = value.join(',');
				}
				Object.assign(vm.ipsecCertQueryParams, params);
			},
			 // 清除筛选
			clearFilterClick(){
				var vm = this,
					params = {};
				
				vm.advancedQueryItemList.map((items)=>{
					if(items.isShow && items.isShow== true ){
						if(items.type == 'checkbox'){
							items.checkedItemList = [];
							items.oldCheckedItemList = [];
							items.checkAll = false;
							items.isIndeterminate = false;
						}else if(items.type == 'select'){
							items.selectVal = '';
						}
						if(items.type == 'checkbox' || items.type == 'select'){
							params[items.value] = '';	                        
						}
					}
				})
				Object.assign(vm.ipsecCertQueryParams, params);
				document.body.click();
			},
			
	    	//------------------------------------------------------------ 已选 ----------------------------------------------------------    	
	    	// 打开已选弹窗
            openBulkSelectTable(){
                var vm = this;
                
                vm.bulkSelectShow = true;
            },
            // 关闭已选弹窗
            closeBulkSelectTable(){
                var vm = this;
                
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 清空事件
            clearBulkSelected(){
                var vm = this;
				
                if(vm.activeName == 'ipsecCertificate'){
                	vm.$refs["ipsecCertCtable"].clearSelection();
                }else if(vm.activeName == 'sslCertificate'){
                	vm.$refs["sslCertTable"].clearSelection();
                }
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            delBulkSelected(rows){
				var vm = this,
					activeName = vm.activeName,
					tabsCodes = {
						'ipsecCertificate':'ipsecCertCtable',
						'sslCertificate':'sslCertTable',
					},
					rowKeyCodes = {
						'ipsecCertificate':'serialNumber',
						'sslCertificate':'serialNumber',
					},
					tabs = tabsCodes[activeName],
					rowKey = rowKeyCodes[activeName];
				vm.certSelectData = vm.certSelectData.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);
				if(vm.certSelectData.length == 0){
                	vm.bulkSelectShow = false;
                }
            }
	    },
		computed:{
			limitBatch(){
				return batchOperation ? '' : 1;
			},
			hasLicenseRole() {
				return  writableMap.CODE_IPSEC_CERT == true;
			},
            optionalSslCertList() {
                var vm = this,
                    selectType = vm.sendSSLCertForm.productType;
                if(selectType){
                    return vm.sslCertList.filter(item => item.productType.includes(selectType));
                }else{
                    return [];
                }
            },
		},
	    mounted(){
	    	this.init();
		}
	});
</script>