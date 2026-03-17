<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
	#gnbIpsecCertManagePage{
		height: 100%;
		display: flex;
		width:100%;
	}
	#gnbIpsecCertManagePage .el-card__body {
		background: #fff;
		border: none;
		/*padding: 20px;*/
	}
	#gnbIpsecCertManagePage .el-card__footer {
		display: none;
	}
	#gnbIpsecCertManagePage .el-input__suffix {
		top: 4px;
	}
	#gnbIpsecCertManagePage .el-textarea__inner {
		line-height: 1.2;
		padding: 2px 15px;
		resize: none;
	}
	#gnbIpsecCertManagePage .el-upload__tip {
		margin-top: 0;
	}
	#gnbIpsecCertManagePage .slide-content{
		margin-left:0 !important;
	}
	#gnbIpsecCertManagePage .el-dialog__body{
		padding:24px 20px 20px;
		min-height:98px;
	}
	#gnbIpsecCertManagePage .el-dialog__footer{
		padding:20px 20px 26px;
		text-align:left;
	}
	#gnbIpsecCertManagePage .el-checkbox-group{
		padding:16px 0 10px;
	}
	#gnbIpsecCertManagePage .w320{
		width:320px;
	}
	#gnbIpsecCertManagePage .issueStatus{
		margin-top:4px;
		font-size:17px;
		margin-right:10px;
		float:left;
	}
	#gnbIpsecCertManagePage .issueNoColor:before{
		color:#CFCFCF;
	}
	#gnbIpsecCertManagePage .issueOkColor:before{
		color:#67D972;
	}
	#gnbIpsecCertManagePage .statusTip{
		overflow:hidden;
		text-overflow:ellipsis;
		white-space:nowrap;
		font-size:12px !important;
	}
	#gnbIpsecCertManagePage .el-checkbox__input.is-checked+.el-checkbox__label{
		color:#333;
		font-weight:normal;
	}
	#gnbIpsecCertManagePage .caCertList .el-form-item:nth-child(2) .el-form-item__label{
		font-size:12px;
		padding-bottom:4px;
	}
	#gnbIpsecCertManagePage .snTableBorder{
		border:1px solid #DFE2EE;
	}

	#gnbIpsecCertManagePage .issueErrorColor:before,
	#gnbIpsecCertManagePage .statusError:before{
		color:#E88282;
	}

	#gnbIpsecCertManagePage .tableBackup{
		font-size:12px;
		color:#4D84FF;
		cursor:pointer;
		text-decoration:underline;
	}
	#gnbIpsecCertManagePage .backupUpload{
		margin-left:5px;
		float:left;
		width:81%;
		overflow:hidden;
		text-overflow:ellipsis;
		white-space:nowrap;
	}
	/* 最新样式表 */
	#gnbIpsecCertManagePage .el-form-item{
		margin-bottom:26px !important;
	}

	#gnbIpsecCertManagePage .el-form-item__content{
		line-height:26px;
	}
	#gnbIpsecCertManagePage .taskInput{
		width:172px;
	}
	#gnbIpsecCertManagePage .el-message-box{
		padding-bottom:20px;
	}
	#gnbIpsecCertManagePage .el-message-box__btns > .el-button:not(:first-child){
   		 margin: 0 15px 0 0;
	}
	#gnbIpsecCertManagePage .el-message-box__content{
		padding:30px;
	}
	#gnbIpsecCertManagePage .el-message-box__message p {
		margin:0 !important;
	}
	#gnbIpsecCertManagePage .el-tooltip__popper{
		padding:10px;
	}
	#gnbIpsecCertManagePage .statusProgress{
		font-size:12px !important;
	}
	#gnbIpsecCertManagePage .batchDeleteWarp .el-dialog__body{
		min-height:60px !important;
	}
	#gnbIpsecCertManagePage .el-input-group__append {
		padding: 0 6px;
	}
	#gnbIpsecCertManagePage .el-input-group__append .el-icon:before { color: #7A7992; }

	#gnbIpsecCertManagePage .issueInfo{
		margin-right:8px;
		font-size:18px;
	}
	#gnbIpsecCertManagePage .issueInpro{
		margin-right:8px;
		font-size:18px;
	}
	#gnbIpsecCertManagePage .commonQuery .pairgrid-query {
		width: 400px !important;
	}
	#gnbIpsecCertManagePage .el-tabs__active-bar.is-top{
		bottom: 0 !important;
	}
	#gnbIpsecCertManagePage .el-tabs {
		width: 100%;
		height: 100%;
	}
	#gnbIpsecCertManagePage .commonTabsTop .el-tabs__header {
		border-bottom: 1px solid #D5DCEC;
	}
	#gnbIpsecCertManagePage .el-input-group__append {
		padding: 0 6px;
	}
	#gnbIpsecCertManagePage .jumpSSLCertIcon:hover::after {
		margin-left: -55px;
	}
</style>
<!--gnb设备证书页面  -->
<div class="overflow-cls">
	<div id="gnbIpsecCertManagePage" class='commonFlex' style="min-width: 900px;position: relative;overflow-y: hidden;">
		<div class='leftWarp commonWarp'>
			<div v-if="gnbActiveName == 'gnbIpsecCert'">
				<div class="circleIcon placeholder-bt" style="right:50px;top:42px" placeholder='<%=rb.getString("CAZhengShu")%>'>
					<span class="el-icon el-icon-circle-certificate" @click="gnbCACertificate"></span>
				</div>
				<div v-show="optBtnShow" class="circleIcon placeholder-bt" style="right: 15px;top:42px" placeholder='<%=rb.getString("ZhengShuDaoRu")%>'>
					<span class="el-icon el-icon-circle-import" @click="gnbImportCertificate"></span>
				</div>
			</div>
			<!-- gNB SSL 证书操作项 -->
			<div v-if="gnbActiveName == 'gnbSslCert'">
				<!-- 全部设备下发证书 -->
				<div v-show="optBtnShow" class="circleIcon placeholder-bt" placeholder='<%=rb.getString("XiaFaZhengShu")%>'style='right: 52px;top:42px'>		
					<span class="el-icon el-icon-status-issued" @click="gnbAllDeviceIssuedCertificate" style='font-size: 14px;'></span>
				</div>
				<!-- SSL 证书维护 -->
				<div class="circleIcon placeholder-bt jumpSSLCertIcon" placeholder='SSL Certificate' style='top:42px'>		
					<span class="el-icon el-icon-circle-certificate" @click="gnbJumpSslCertificatePage"></span>
				</div>
			</div>
			<!--主体内容-->
			<el-tabs v-model="gnbActiveName" @tab-click='gnbTabClick' class='commonTabsTop'>			 	
				<!-- 4g Ipsec Certificate :url="ipsecCertUrl"-->
				<el-tab-pane label='<%=rb.getString("OMCZhengShuGuanLi")%>' name="gnbIpsecCert">
					<el-ctable ref="gnbCertCtable" :time="6" :url="gnbCertManageUrl" :query-params="gnbCertManageParams" id="gnbCertCtable" :limit="limitBatch"
						:page-size="pageSize" pagination="true" :rownumber=true :row-key="'serialNumber'" @selection-change='gnbBatchSelect'>
						<template slot="toolbar">
							<div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
								<!-- 已选数据 -->
								<div v-show="optBtnShow" class="selectBlukBoxCls">
									<div class="selectMain headBtnItemCls">
										<div class="bulkSelectBtnBoxCls" @click="gnbOpenBulkSelectTable" style='border-right: 0; padding: 0;'>
											<span class="el-icon-selected el-icon"></span>
											<span class="bulkSelectNumBoxCls">( {{certSelectData.length}} )</span>
										</div>
										<div class="selectTableBoxCls" v-show="bulkSelectShow" style="position: absolute;top: 32px;left: 30px;">
											<div class="selectBoxTitle">
												<span><%=rb.getString("YiXuan")%></span>
												<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="gnbCloseBulkSelectTable"></span>
											</div>
											<div class="selectBoxMain">
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div><%=rb.getString("XiaoZhanBianMa")%></div>
														<div @click="gnbClearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
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
																	<span @click="gnbDelBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</div>
										</div>
									</div>
								</div>
								<!-- 当同时执行下发证书、备份证书、更新证书时，则按照时间顺序执行，第一个任务执行结束后执行第二个任务； -->
								<div v-show="optBtnShow" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbIssueBatch">
									<span class='el-icon el-icon-status-issued'></span>
									<span><%=rb.getString("XiaFaZhengShu")%></span>
								</div>
								<div v-show="optBtnShow" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbBackupBatch">
									<span class='el-icon el-icon-operation-backups'></span>
									<span><%=rb.getString("ZhengShuBeiFen")%></span>
								</div>
								<div v-show="optBtnShow" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbDownloadBatch">
									<span class='el-icon el-icon-operation-download'></span>
									<span><%=rb.getString("PiLiangXiaZai")%></span>
								</div>
								<div v-show="optBtnShow" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbDeleteBatch" style='border-right: 0;'>
									<span class='el-icon el-icon-operation-delete'></span>
									<span><%=rb.getString("PiLiangShanChu")%></span>
								</div>
							</div>
							<div class='toolbarHeadBtnBoxCls commonQuery tableHeadQueryBoxCls' style=' border: 0; padding: 5px 0;'>
								<el-query type="normal" @query="gnbQueryIpsec" placeholder="<%=rb.getString("ZhengShuSouSuo")%>"></el-query>
							</div>
						</template>
						<el-table-column v-if="optBtnShow" type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column width="40">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-info" @click="gnbOptInfoClick(scope.row,event)"></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serialNumber" show-overflow-tooltip></el-table-column>
						<el-table-column label="<%=rb.getString("CAZhengShu")%>" prop="caFileName" show-overflow-tooltip >
							<!-- CA证书当前状态 : 0 == 下发失败，1 == 已下发，2 == 下发中，3 == 未下发 -->
							<template slot-scope="scope">
								<div v-if="scope.row.caUploadStatus == '0'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuXiaFaShiBai")%>" placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueErrorColor"></span>
									</el-tooltip>
									{{scope.row.caFileName}}
								</div>
								<div v-if="scope.row.caUploadStatus == '1'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuYiXiaFa")%>" placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueOkColor"></span>
									</el-tooltip>
									{{scope.row.caFileName}}
								</div>
								<div v-if="scope.row.caUploadStatus == '2'">
									<div class='commonFlex'>
										<el-tooltip content="<%=rb.getString("ZhengShuXiaFaZhong")%>"  placement="bottom">
											<span class="status_issuing" style='margin-top: 5px; margin-right: 10px;'></span>
										</el-tooltip>
										<span class="statusProgress">{{scope.row.caFileName}}</span>
									</div>
								</div>
								<div v-if="scope.row.caUploadStatus == '3' " class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuWeiXiaFa")%>" placement="bottom">
										<span v-if="scope.row.caFileName !=null &&  scope.row.caFileName !=''" class="el-icon el-icon-status-issued issueStatus issueNoColor" ></span>
									</el-tooltip>
									{{scope.row.caFileName}}
								</div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("IpsecZhengShu")%>" prop="certFileName" show-overflow-tooltip>
							<!-- Ipsec证书当前状态: 0 == 下发失败，1 == 已下发，2 == 下发中，3 == 未下发-->
							<template slot-scope="scope">
								<div v-if="scope.row.certUploadStatus == '0'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuXiaFaShiBai")%>" placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueErrorColor"></span>
									</el-tooltip>
									{{scope.row.certFileName}}
								</div>
								<div v-if="scope.row.certUploadStatus == '1'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuYiXiaFa")%>" placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueOkColor"></span>
									</el-tooltip>
									{{scope.row.certFileName}}
								</div>
								<div v-if="scope.row.certUploadStatus == '2'">
									<div class='commonFlex'>
										<el-tooltip content="<%=rb.getString("ZhengShuXiaFaZhong")%>"  placement="bottom">
											<span class="status_issuing" style='margin-top: 5px; margin-right: 10px;'></span>
										</el-tooltip>
										<span class="statusProgress">{{scope.row.certFileName}}</span>
									</div>
								</div>
								<div v-if="scope.row.certUploadStatus == '3'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuWeiXiaFa")%>" placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueNoColor"></span>
									</el-tooltip>
									{{scope.row.certFileName}}
								</div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("MiYaoZhengShu")%>" prop="secretKeyFileName" show-overflow-tooltip>
							<!-- 秘钥证书当前状态  0 == 下发失败，1 == 已下发，2 == 下发中，3 == 未下发 -->
							<template slot-scope="scope">
								<div v-if="scope.row.secretKeyUploadStatus == '0'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuXiaFaShiBai")%>" placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueErrorColor"></span>
									</el-tooltip>
									{{scope.row.secretKeyFileName}}
								</div>
								<div v-if="scope.row.secretKeyUploadStatus == '1'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuYiXiaFa")%>"  placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueOkColor"></span>
									</el-tooltip>
									{{scope.row.secretKeyFileName}}
								</div>
								<div v-if="scope.row.secretKeyUploadStatus == '2'">
									<div class='commonFlex'>
										<el-tooltip content="<%=rb.getString("ZhengShuXiaFaZhong")%>"  placement="bottom">
											<span class="status_issuing" style='margin-top: 5px; margin-right: 10px;'></span>
										</el-tooltip>
										<span class="statusProgress">{{scope.row.secretKeyFileName}}</span>
									</div>
								</div>
								<div v-if="scope.row.secretKeyUploadStatus == '3'" class="statusTip">
									<el-tooltip content="<%=rb.getString("ZhengShuWeiXiaFa")%>"  placement="bottom">
										<span class="el-icon el-icon-status-issued issueStatus issueNoColor"></span>
									</el-tooltip>
									{{scope.row.secretKeyFileName}}
								</div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("ZhengShuBeiFen")%>" prop="backupFileName" show-overflow-tooltip>
							<template slot-scope="scope">
								<!-- 证书备份：0 == 未备份，1 == 已备份，2 == 备份中 -->
								<div v-if="scope.row.backupUploadStatus == '1'">
									<div class="">
										<a class="tableBackup" @click="gnbBackupDownload(scope.row.id)"> {{scope.row.backupFileName}}</a>
									</div>
								</div>
								<div v-if="scope.row.backupUploadStatus == '2'" class="statusProgress">
									<span class="status_backup" style="float:left;"></span>
									<span class="backupUpload"><%=rb.getString("ZSBeiFenZhong")%></span>
								</div>
							</template>
						</el-table-column>
					</el-ctable>
				</el-tab-pane>

				<!-- 5g-SSL 证书  :url="gnbSslCertificateUrl" :data='sslCertData' -->
				<el-tab-pane label='<%=rb.getString("TRSSLZhengShu")%>' name="gnbSslCert">
					<el-ctable ref="gnbSslCertTable" :time="6" :url="gnbSslCertificateUrl" :query-params="gnbSslCertificateParams" id="gnbSslCertTable" :limit="limitBatch"
					   :page-size="pageSize" pagination="true" :rownumber=true :row-key="'serialNumber'" @selection-change='gnbSslCertBatchSelect'>
					   <template slot="toolbar">
					   <div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
								<!-- 已选数据 -->
								<div v-show="optBtnShow" class="selectBlukBoxCls">
									<div class="selectMain headBtnItemCls" style='border-right: 0; padding: 0;'>
										<div class="bulkSelectBtnBoxCls" @click="gnbOpenBulkSelectTable">
										   <span class="el-icon-selected el-icon"></span> 
										   <span class="bulkSelectNumBoxCls">( {{certSelectData.length}} )</span>
									   </div>
									   <div class="selectTableBoxCls" v-show="bulkSelectShow" style="position: absolute;top: 32px;left: 30px;">
											<div class="selectBoxTitle">
												<span><%=rb.getString("YiXuan")%></span>
												<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="gnbCloseBulkSelectTable"></span>
											</div>
											<div class="selectBoxMain">
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div><%=rb.getString("XiaoZhanBianMa")%></div>
														<div @click="gnbClearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
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
																	<span @click="gnbDelBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
																</div>
															</template>
														</el-table-column>
													</el-ctable>
												</div>
											</div>
										</div>
									</div>
							    </div>
								<div v-show="optBtnShow" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbSingleDeviceIssuedCertificate" style='border-right: 0;'>
									<span class='el-icon el-icon-status-issued'></span>
									<span><%=rb.getString("FaSongZhengShu")%></span>
								</div> 
							</div>
							<div class='commonQuery' style=' border: 0; margin-top: 10px;'>
								<el-query type="normal" @query="gnbQuerySSLCertificate" placeholder="<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("SSLZhengShu")%>"></el-query>
							</div>
					   </template>
					   <el-table-column v-if="optBtnShow" type="selection" :reserve-selection="true"></el-table-column>
					   <el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serialNumber" show-overflow-tooltip></el-table-column>   
					   <el-table-column label="<%=rb.getString("SSLZhengShu")%>" prop="sslCertName" show-overflow-tooltip >
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
					   <el-table-column label="<%=rb.getString("YouXiaoKaiShiShiJian")%>" prop="validStartTime" show-overflow-tooltip></el-table-column>
					   <el-table-column label="<%=rb.getString("YouXiaoJieShuShiJian")%>" prop="validEndTime" show-overflow-tooltip></el-table-column>	
					   <el-table-column label="<%=rb.getString("FaSongShiJian")%>" prop="updateTime" show-overflow-tooltip></el-table-column>	
				   </el-ctable>           
			   </el-tab-pane>
			</el-tabs>
		</div>

		<!-- 导入文件框 -->
		<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='showImportCard'>
			<div class='rightWarpLayer'>
		 		<div class='rightBoxHeaderHasTip'>
					<div class='headerText'>
						<span><%=rb.getString("ZhengShuDaoRu")%> <%=rb.getString("ipsecCert")%></span>
						<span class='closeIconBox' @click='gnbCloseImportDevice'><i class='el-icon el-icon-close'></i></span>
					</div>
				</div>
				<div class='rightWarpLayerContent'>
					<el-form label-position="top" ref="ruleForm" :model='ruleForm' :rules='gnbRules' style='padding: 30px 20px;'>
		                <el-form-item label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serialNumber'>
							<el-input maxlength="50" v-model="ruleForm.serialNumber" class="taskInput" @change="gnbInputChange" style='width: 320px;'>
								<div slot="append">
									<span class="el-icon el-icon-common-select commonTemplateText12" @click="gnbSelectDevice"> <%=rb.getString("XuanZeYiYouSheBei")%></span>
								</div>
							</el-input>
						</el-form-item>
						<el-form-item v-show="snTableSelect">
		                    <el-ctable ref="importDeviceTable" id='importDeviceTable' :time='6' :url="snTableUrl" :query-params="serialNumParams" :row-key="'serialNumber'" style='border-radius: 4px; border: 1px solid #DFE2EE; margin: 0;'
				            	:show-pager="false" :page-size="pageSize" pagination="false" :rownumber=false height="330px"
				            	@row-click="gnbSnChange">
				               <template slot="toolbar">
				               		<div class="searchCon" style="margin-left:18px;">
										<el-input placeholder="<%=rb.getString("XiaoZhanBianMa")%>" suffic-icon='el-icon-search' v-model="snSearchText" style="width:270px;">
											<i slot="suffix" class="el-icon el-icon-common-search"  @click="gnbQuerySnSearch"></i>
										</el-input>
									</div>
				                </template>
				               	<el-table-column width="50">
									<div slot-scope="scope" style="margin: 0 auto;">
										<el-radio v-model="serialNumberRadio" :label="scope.row.serialNumber"><span></span></el-radio>
									</div>
								</el-table-column>
				                <el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serialNumber"></el-table-column>
				                <!-- 新接口无此字段 -->
				                <el-table-column label="<%=rb.getString("HostName")%>" prop="host_name" v-if='false'></el-table-column>
				            </el-ctable>
	                    </el-form-item>
	                    <el-form-item label="<%=rb.getString("IpsecZhengShu")%>" prop='certsName'>
	                        <el-upload :on-success='gnbCheckFile' :on-change="gnbFileChange" :show-file-list=false ref="uploadCert"
	                        	:action="ruleForm.uploadFileUrl" :data="fileParams" name="uploadCertFile" :auto-upload="false">
	                            <el-input :readonly="true" :value=certsName class="w320">
									<a slot="append" class="el-icon el-icon-operation-import" @click="gnbFileSelect"></a>
								</el-input>
	                            <a slot="trigger" ref="file_up"></a>
	                        </el-upload>
                    	</el-form-item>
                    	<el-form-item label="<%=rb.getString("MiYaoZhengShu")%>" prop='secretKeyName'>
	                        <el-col :span="10">
	                            <el-upload :on-success='gnbCheckFile2' :on-change="gnbFileChangePrivateKey" :show-file-list=false ref="uploadCert"
	                            	:action="ruleForm.uploadFileUrl" name="uploadSecretKeyFile"
	                                :data="fileParams" :auto-upload="false">
	                                <el-input :readonly="true" :value=secretKeyName class="w320">
	                                    <a slot="append" class="el-icon el-icon-operation-import" @click="gnbFileSelectPrivateKey"></a>
	                                </el-input>
	                                <a slot="trigger" ref="file_up2"></a>
	                            </el-upload>
	                        </el-col>
                    	</el-form-item>
	                    <el-form-item label="<%=rb.getString("ZhengShuMiaoShu")%>" prop='description'>
	                        <el-input v-model='ruleForm.description' type='textarea' :rows='4' class="w320"></el-input>
	                    </el-form-item>
					</el-form>
				</div>
				<div class='commonFlex commonBorderTop commonFormFotter'>
					<div>
						<el-button type="primary" @click="gnbUploadDevice"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="gnbCloseImportDevice"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</div>
			</div>
		</div>

		<!-- 证书详情 -->
		<div class='rightWarp' style='position: relative; flex: 0 1 400px;' v-show='showCertInfoDialog'>
			<div class='rightWarpLayer'>
		 		<div class='rightBoxHeaderHasTip'>
					<div class='headerText'>
						<span><%=rb.getString("XinXi")%></span>
						<span class='closeIconBox' @click=gnbCloseCertInfoDialog><i class='el-icon el-icon-close'></i></span>
					</div>
					<div class='commonTemplateText14' style='padding-top: 10px;'>
						<%=rb.getString("XiaoZhanBianMa")%>: {{infoSerialNumber}}
					</div>
				</div>
				<div class='rightWarpLayerContent'>
					<div class='infoWarp' style='border-top: none;'>
						<p class="commonText14"><%=rb.getString("CAZhengShu")%></p>
						<div class="info" style='margin-top: 16px; position: relative;'>
							<label><%=rb.getString("CAZhengShu")%></label>
							<div class="commonSize14" v-html="caFileName" style='position: absolute; right: 0; display: inline-flex;'></div>
						</div>
						<div class="info">
							<label><%=rb.getString("ShangChuanRen")%></label>
							<span>{{caUploader}}</span>
						</div>
						<div class="info">
							<label><%=rb.getString("ShangChuanShiJian")%></label>
							<span>{{caUploadTime}}</span>
						</div>
						<div class="info">
							<label><%=rb.getString("XiaFaShiJian")%></label>
							<span>{{caDownTime}}</span>
						</div>
						<div class="info">
							<label><%=rb.getString("WangGuanMiaoShu")%></label>
							<p class="description commonSize14">{{caDescription}}</p>
						</div>
					</div>
					<div class='infoWarp'>
						<p class="commonText14"><%=rb.getString("SheBeiZhengShu")%></p>
						<div class="info" style='margin-top: 16px; position: relative;'>
							<label><%=rb.getString("IpsecZhengShu")%></label>
							<div class="commonSize14" v-html="certFileName" style='position: absolute; right: 0; display: inline-flex;'></div>
						</div>
						<div class="info" style='position: relative;'>
							<label><%=rb.getString("MiYaoZhengShu")%></label>
							<div class="commonSize14" v-html="secretKeyFileName" style='position: absolute; right: 0; display: inline-flex;'></div>
						</div>
						<div class="info">
							<label><%=rb.getString("ShangChuanRen")%></label>
							<span>{{privateUploader}}</span>
						</div>
						<div class="info">
							<label><%=rb.getString("ShangChuanShiJian")%></label>
							<span>{{privateUploadTime}}</span>
						</div>
						<div class="info">
							<label><%=rb.getString("XiaFaShiJian")%></label>
							<span>{{privateDownTime}}</span>
						</div>
						<div class="info">
							<label><%=rb.getString("ZhengShuMiaoShu")%></label>
							<p class="description commonSize14">{{privateDescription}}</p>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!--证书批量下发-->
		<el-dialog title='<%=rb.getString("ZhengShuXiaFa")%>' :visible="showConfirmInfoBatch" width="1000" class="caCertList"
			:close-on-click-modal="false" :modal-append-to-body="false" @close="gnbCancelgnbIssueBatch">
			<el-form :model="confirmFormBatch" ref="confirmFormBatch" label-position="top">
				<el-form-item label="<%=rb.getString("QueDingPiLiangXiaFaCiZhengShuMa")%>" >
					<el-checkbox v-model="caCertEnableBatch" checked label="<%=rb.getString("CAZhengShu")%>"></el-checkbox>
					<el-checkbox v-model="deviceCertEnableBatch" checked label="<%=rb.getString("SheBeiZhengShuIpsecZhengshuMiYaoZhengShu")%>"></el-checkbox>
					<div slot="tip" class="el-upload__tip" v-show="selectBatchIssueFlag"><%=rb.getString("CAZhengShuSheBeiZhengShuBiXuanQiYi")%></div>
				</el-form-item>
				<el-form-item label="<%=rb.getString("CAZhengShu")%>" v-show="caTableSelectBatch" style="margin-bottom:0px !important;">
					<el-ctable ref="issueCertTableBatch" :url="batchIssueCaCertUrl" :query-params="issueCertParamsBatch" class="snTableBorder"
		            	:page-size="pageSize" pagination="true" :rownumber=false :row-key="'id'" height="330px"
		            	@row-click="gnbIssueChangeBatch">
		                <template slot="toolbar">
		                	<div class="searchCon" style="margin-left:18px;">
								<el-input placeholder="<%=rb.getString("ZhengShuWenJian")%>" suffic-icon='el-icon-search' v-model="batchCertsSearchText" style="width:270px;">
									<i slot="suffix" class="el-icon el-icon-common-search"  @click="gnbIssueCertQueryBatch"></i>
								</el-input>
							</div>
		                </template>
		               	<el-table-column width="50">
							<div slot-scope="scope" style="margin: 0 auto;">
								<el-radio v-model="caFileIdBatch" :label="scope.row.id"><span></span></el-radio>
							</div>
						</el-table-column>
		                <el-table-column label="<%=rb.getString("ZhengShuWenJian")%>" prop="fileName"></el-table-column>
		                <el-table-column label="<%=rb.getString("ZhengShuMiaoShu")%>" prop="description"></el-table-column>
		            </el-ctable>
		            <div v-show="selectBatchCaList">
		            	<div slot="tip" class="el-upload__tip" ><%=rb.getString("QingXuanZeYaoXiaFaDeCAZhengShu")%></div>
		            </div>
				</el-form-item>
			</el-form>
			<div slot="footer">
				<div class="buttonGroup">
					<el-button type="primary" @click="gnbConfirmIssueBacth"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gnbCancelgnbIssueBatch"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</el-dialog>

		<el-dialog title="<%=rb.getString("QueRen")%>" :visible="gnbShowBatchDeleteInfo" width="420" class="batchDeleteWarp"
			:close-on-click-modal="false" :modal-append-to-body="false" @close="gnbShowBatchDeleteInfo = false">
			<div><%=rb.getString("QueDingPiLiangShanChuZhengShu")%></div>
			<div style="padding-top:10px;"><%=rb.getString("JinXingZhongDeRenWuBuKeYiShanChu")%></div>
			<span slot="footer" class="dialog-footer">
				<div style="text-align:right;">
					<el-button type="primary" @click="gnbBacthDeleteConfirm"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gnbShowBatchDeleteInfo = false"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</span>
		</el-dialog>

		<!-- 跳转到CA 证书管理页面 -->
		<el-slide ref="slideCA" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
			:width="slideWidth" @cancel='gnbCancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>

		<!--SSL 证书 给全部设备下发证书-->
		<el-dialog title='<%=rb.getString("FaSongZhengShu")%>' :visible.sync="gnbShowSendSSLCertificate" width="500" class="sendCertFormWarp"
			:close-on-click-modal="false" @close="gnbCancelSendCertificate">
			<el-form :model="gnbSendSSLCertForm" ref="gnbSendSSLCertForm" :rules="gnbSendSSLCertRule" label-position="left" label-width="110">
				<span><%=rb.getString("ShiFouXiaFaZhengShuDaoOMC")%></span>
				<el-form-item label="<%=rb.getString("SSLZhengShu") %>" prop="sslCert" placeholder="<%=rb.getString("QingXuanZe") %>" style='padding-top: 20px;'>
	                <el-select v-model="gnbSendSSLCertForm.sslCert">
	                    <el-option v-for="item in gnbSslCertList" :label="item.text" :value="item.value"></el-option>
	                </el-select>
	            </el-form-item>
			</el-form>		
			<div slot="footer">
				<div class="buttonGroup">
					<el-button type="primary" @click="gnbConfirmSendCertificate"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gnbCancelSendCertificate"><%=rb.getString("QuXiao")%></el-button>
				</div>	
			</div>
		</el-dialog>
		<el-slide ref="gnbSlideCertManagePage" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition" class='jumpManagePageSlide'
			:width="slideWidth" @cancel='gnbSslcancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
	</div>
</div>
<script type="text/javascript">
	var gnbCertificateManageVue = new Vue({
	    el: '#gnbIpsecCertManagePage',
	    data() {
			var vm = this,

	    		serialNumberValidator = (rule, value, callback) => {
                	if (value === '' || value === null || value === undefined) {
	                	callback('<%=rb.getString("UPSSNBuNengWeiKong")%>');
	                } else {
	                	callback()
	                }
	            },
		        validateCertsName = function(rule,value,callback) { // 校验设备
		           	value = vm.certsName;
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else {
						callback();
					}
				},
				validateSecretKeyName = function(rule,value,callback) { // 校验设备
		           	value = vm.secretKeyName;
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else {
						callback();
					}
				},
				validatorIpAddress = (rule,value,callback) => {
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("IPDiZhiBuNengWeiKong")%>'))
					}else{
						callback();
					}
				},

				validateSSLCert = (rule,value,callback) => {				
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingXuanZeSSLZhengShu")%>'))
					}else{
						callback();
					}
				};
	    	return {
				gnbActiveName: 'gnbIpsecCert',
				//-------------------------------------------------SSL Cert
				gnbSslCertificateUrl: '${ctx}/cert/ssl/getGNBInfoCertPageList.action',
				gnbSslCertificateParams: {
	    			timeZone: timeZone,
	             	searchText: ''
	            },
				// send certificate
				gnbShowSendSSLCertificate: false,
				gnbSendSSLCertForm: {
					sslCert: ''
				},
				gnbSslCertList: [],
				gnbSendSSLCertRule: {
					sslCert: [{validator: validateSSLCert}],
				},
				sendSSLCertFlag: 'all',

				//-------------------------------------------------IPSEC Cert
				gnbShowBatchDeleteInfo: false,
	    		pageSize: 50,
	    		gnbCertManageParams: {
	    			timeZone: timeZone,
	             	searchText: '',
	             	nbType: 'gNB'
	            },
	            snSearchText: '',
	            serialNumParams: {
	            	timeZone: timeZone,
	             	searchText: '',
	             	nbType: 'gNB'
	            },

	            batchCertsSearchText: '',
	            issueCertParamsBatch: {
	            	timeZone: timeZone,
	             	searchText: '',
	             	nbType: 'gNB'
	            },
	            gnbCertManageUrl: '',
	            batchIssueCaCertUrl: '',
	            snTableUrl: '${ctx}/cell/ipsec/cert/queryCellCertList.action',
	            certSelectData: [],
	            showImportCard: false,
	            serialNumberRadio: '',
	            //导入设备证书
	            ruleForm: {
	                uploadFileUrl: "${ctx}/cell/cert/uploadIpsecPrivateCertFile.action",
	                serialNumber: '',//基站编码
	               	description: '',//描述
	               	secretKeyName: '',//秘钥文件为空
	               	certsName: '' //设备证书文件为空
	           	},
	           	gnbRules: {
	           		serialNumber: [ {required: true, validator: serialNumberValidator} ],
                    certsName: [ {required: true, validator: validateCertsName} ],
                    secretKeyName: [ {required: true, validator: validateSecretKeyName} ],
	            },
	            fileParams: {},
	            certsName: '',
				secretKeyName: '',

				slideUrl: '',
				slideTitle: '',
				slideHeader: false,
				slideFooter: false,
				slidePosition: 'top',
				slideModal: false,
				slideWidth: '',
				slideHeight: '',

				selectBatchIssueFlag: false,
				selectBatchCaList: false,
				showConfirmInfoBatch: false,

				confirmFormBatch: {},
				caFileIdBatch: '',
				snTableSelect: false,
				caCertEnableBatch: true,
				deviceCertEnableBatch: true,
				caTableSelectBatch: true,
				caCurEnable: '',
				ipsecCurEnable: '',
				caCurEnableBatch: '',
				ipsecCurEnableBatch: '',
	            rowData: [],
	            batchDeleteIds: [],
				//详情
				showCertInfoDialog: false,
				infoSerialNumber: '',
				caFileName: '',
				caUploader: '',
				caUploadTime: '',
				caDownTime: '',
				caDescription: '',
				certFileName: '',
				secretKeyFileName: '',
				privateUploader: '',
				privateUploadTime: '',
				privateDownTime: '',
				privateDescription: '',

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
						value: 'certStatus'
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
						value: 'updateStatus'
					}
				],
	    	}
	    },
	    computed: {
			limitBatch(){
				return batchOperation ? '' : 1;
			},
			optBtnShow() {
				return writableMap['CODE_GNB_IPSEC_CERT'] == true;
			},
	    },
	    watch: {
			//ca 批量证书勾选
	    	caCertEnableBatch(val){
	    		var vm = this;

	    		vm.confirmFormBatch.caCertEnableBatch = val == '1' ? '1':'0';

	    		if(val){ //true
	    			vm.caTableSelectBatch = true;
	    		}else{
	    			vm.caTableSelectBatch = false;
	    			//清空已选择项
	    			vm.caFileIdBatch = '';
	    		}
			},
			//设备批量证书勾选
	    	deviceCertEnableBatch(val){
	    		var vm = this;

	    		vm.confirmFormBatch.deviceCertEnableBatch = val == '1' ? '1':'0';
			},
		},
	    methods: {
	    	gnbInit(){
	    		var vm = this;

	    		vm.gnbCertManageUrl = "${ctx}/cell/cert/getIpsecCertInfoPageList.action";
	    	},
			//tab 切换                                                                     
			gnbTabClick(tab){
				var vm = this;		
				
				vm.$refs['gnbCertCtable'].clearSelection();
				vm.$refs["gnbSslCertTable"].clearSelection();
				vm.bulkSelectShow = false;
				vm.showSettingDialog = false;
				//详情关闭
				vm.showCertInfoDialog = false;
				//导入关闭
				vm.showImportCard = false;
			},
	    	/**
			* 列表选中
			* @param selection{Array}   选中数据
			*/
			gnbBatchSelect(selection){
				var vm = this;

			    vm.certSelectData = selection.map((item)=>{
					return Object.assign(item,{snNumber:item.serialNumber})
				});
			},

			//表格操作-备份点击下载文件
			gnbBackupDownload(id){
				var vm = this;

		    	exportByForm("${ctx}/cell/cert/downLoadIpsecCertBackupFile.action",{
		        	id: id,
		        	nbType: 'gNB'
		        });
			},
	    	//Ipsec 模块  搜索事件
	    	gnbQueryIpsec(val){
				var vm = this;

	    		Object.assign(vm.gnbCertManageParams,{
	    			searchText: val
	    		})
	    	},

	    	//-------------------------------------------------------------ipsec 证书 导入--------------------------------------------
			//导入设备证书点击事件
			gnbImportCertificate(){
				var vm = this;

				vm.serialNumber = '';
				vm.serialNumberRadio = '';
				vm.$refs.ruleForm.resetFields();
				vm.$refs.gnbCertCtable.clearSelection();//清空勾选的数据
				vm.showImportCard = true;
				vm.snTableSelect = false;
				vm.showSettingDialog = false;
				vm.showCertInfoDialog = false;
			},
	    	//点击选择已有设备
	    	gnbSelectDevice(){
				var vm = this;

                vm.$refs.importDeviceTable.refresh();
	    		vm.snTableSelect = true;
	    	},
	    	//sn选择已有设备
	    	gnbSnChange(row,old){
	    		var vm = this;

				if(row) {
					vm.serialNumberRadio = row.serialNumber;
					vm.ruleForm.serialNumber = row.serialNumber;
				}
	    	},
	    	//导入-输入的基站编码input 改变时
	    	gnbInputChange(value,newVal){
	    		var vm = this;

	    		vm.snTableSelect = false;
	    		vm.serialNumberRadio = '';
	    		vm.$refs.importDeviceTable.clearSelection();
	    	},

	    	//-----------------------------------ipsec 证书选择-----------------------------------
	    	gnbCheckFile(res, file) {
                var vm = this;

                if (res.success) {
                    if (res.suc_count > 0) {
                        vm.$message({
                            type: 'success',
                            message: '<%=rb.getString("ChengGong")%>'
                        });
                    } else {
                        vm.$message({
                            type: 'warning',
                            message: '<%=rb.getString("ShiBai")%>'
                        });
                    }
                } else {
                    vm.$message({
                        type: 'error',
                        message: res.msg
                    });
                }
                //修改已选择文件状态
                var fileList = vm.$refs.uploadCert.gnbUploadFiles;
                fileList.forEach(function (file) {
                    file.status = 'ready';
                })
            },
            /**
           	  *选择文件后，校验格式，并赋值页面显示
             *@param file：文件名称
             */
            gnbFileChange(file, fileList) {
                var vm = this;

                vm.fileParams.uploadCertFile = file.raw;
                vm.certsName = file.name;
            },
         	// 导入文件按钮
            gnbFileSelect() {
                var vm = this;

                vm.$refs.uploadCert.clearFiles();
                vm.$refs['file_up'].click();
            },

	    	//-----------------------------------秘钥证书-----------------------------------
	    	//发送请求，校验device文件内容
	    	gnbCheckFile2(res, file) {
                var vm = this;

                if (res.success) {
                    if (res.suc_count > 0) {
                        vm.$message({
                            type: 'success',
                            message: '<%=rb.getString("ChengGong")%>'
                        });
                    } else {
                        vm.$message({
                            type: 'warning',
                            message: '<%=rb.getString("ShiBai")%>'
                        });
                    }
                } else {
                    vm.$message({
                        type: 'error',
                        message: res.msg
                    });
                }
                //修改已选择文件状态
                var fileList = vm.$refs.uploadCert.gnbUploadFiles;
                fileList.forEach(function (file) {
                    file.status = 'ready';
                })
            },
            /**
             * PrivateKey 导入监听
             * file:文件
             *fileList：文件列表
             */
             gnbFileChangePrivateKey(file, fileList) {
                 var vm = this;

                 vm.fileParams.uploadSecretKeyFile = file.raw;
                 vm.secretKeyName = file.name;
             },
             gnbFileSelectPrivateKey() {  //PrivateKey 导入文件按钮
                 var vm = this;

                 vm.$refs.uploadCert.clearFiles();
                 vm.$refs['file_up2'].click();
             },
             /*确定导入*/
             gnbUploadDevice() {
                 var vm = this, certsName, secretKeyName, url = '';

                 if(vm.certsName === ''){
                	 certsName = false
                 }else{
                	 certsName = true
                 }
                 if(vm.secretKeyName === ''){
                	 secretKeyName = false
                 }else{
                	 secretKeyName = true
                 }

                 vm.fileParams.certsName = vm.certsName;
                 vm.fileParams.secretKeyName = vm.secretKeyName;
                 vm.fileParams.serialNumber = vm.ruleForm.serialNumber;
                 vm.fileParams.description = vm.ruleForm.description;
                 vm.fileParams.nbType = 'gNB';

                 vm.$refs.ruleForm.validate((valid) => {
                     if (valid && secretKeyName && certsName) {
                         this.gnbUploadFiles(vm.ruleForm.uploadFileUrl, vm.fileParams)
                     }
                 })
			},
            /*
           	* 导入函数
            * url：当前的修改或者添加url
            * params： 所有的from参数
            */
            gnbUploadFiles(url, params, cb) {
                var vm = this,
                    xhr = new XMLHttpRequest(),
                    fmd = new FormData();

                if (params) {
                    for (var key in params) {
                        fmd.append(key, params[key]);
                    }
                }
                xhr.open('post', url);
                xhr.send(fmd);
                xhr.onreadystatechange = function () {
                    if (xhr.response) {
                        let str = xhr.response;
                        var objStrMsg = JSON.parse(str);
                        if (str.indexOf("true") != -1) {
                            vm.$message.success('<%=rb.getString("ChengGong")%>');
                            vm.$refs.gnbCertCtable.refresh();
                            vm.showImportCard = false;
            				vm.certsName = '';
            				vm.secretKeyName = '';
            				vm.serialNumber = '';
            				vm.serialNumberRadio = '';
            				vm.snTableSelect = false;
            				vm.$refs.importDeviceTable.clearSelection();
            				vm.$refs.ruleForm.resetFields();
                        } else {
                        	vm.$message({
                                type: 'error',
                                message:objStrMsg.message
                            });
                        }
                    }
                }
            },
         	// 关闭导入弹出框
			gnbCloseImportDevice(){
				var vm = this;

				vm.showImportCard = false;
				vm.certsName = '';
				vm.secretKeyName = '';
				vm.serialNumber = '';
				vm.serialNumberRadio = '';
				vm.snTableSelect = false;
				vm.serialNumParams.searchText = '';
				vm.snSearchText = '';
				vm.$refs.importDeviceTable.clearSelection();//清空表格勾选的状态
				vm.$refs.ruleForm.resetFields();
			},
			//导入基站编码查询
	    	gnbQuerySnSearch(){
	    		var vm = this;

				vm.serialNumParams.searchText = vm.snSearchText;
	    	},

	    	//-------------------------------------------------------------下发操作--------------------------------------------
		    //选择设备选中事件
	    	gnbIssueChangeBatch(row, old){
				var vm = this;

				if(row) {
					vm.caFileIdBatch = row.id;
					vm.selectBatchCaList = false;
				}
			},
	    	//批量下发证书
		    gnbIssueBatch(){
	    		var vm = this,
    				strNum = Math.random().toString();

	    		if(vm.certSelectData.length == 0) return;
	    		vm.showConfirmInfoBatch = true;
	    		//初始化内容
		    	vm.caCertEnableBatch = true;//CA证书默认勾选
    			vm.caTableSelectBatch = true;//CA证书列表默认展示
    			vm.deviceCertEnableBatch = true;
    			vm.batchIssueCaCertUrl = "${ctx}/cell/cert/getIpsecCACertInfoPageList.action?code="+strNum;
	    	},
		  	//点击确定---批量下发
	    	gnbConfirmIssueBacth(){
	    		var vm = this, certIds=[];

	    		//CA证书，设备证书都未勾选
	    		if(vm.caCertEnableBatch == 0 && vm.deviceCertEnableBatch ==  0){
	    			vm.selectBatchIssueFlag = true;
	    			return false;
	    		}else{
	    			vm.selectBatchIssueFlag = false;
	    		}

				certIds = vm.certSelectData.map(function(item){ return item.id});

	    		var caEnableBatch = vm.caCertEnableBatch;
	    		var ipsecEnableBatch = vm.deviceCertEnableBatch;
	    		var caCertId = vm.caFileIdBatch;
	    		if(caEnableBatch == true){
	    			vm.caCurEnableBatch = "1"
	    				//根据被选中的ca列表id进行提示
	    	    		if(caCertId == '' || caCertId == null || caCertId == undefined){
	    					vm.selectBatchCaList = true;
	    					return false;
	    				}else{
	    					vm.selectBatchCaList = false;
	    				}
	    		}else{
	    			vm.caCurEnableBatch = "0"
	    		}
	    		if(ipsecEnableBatch == true){
	    			vm.ipsecCurEnableBatch = "1"
	    		}else{
	    			vm.ipsecCurEnableBatch = "0"
	    		}

	    		axios.post("${ctx}/cell/cert/sendIpsecCert.action",stringify({
					id: certIds.join(','),
					caCertEnable: vm.caCurEnableBatch,
					deviceCertEnable: vm.ipsecCurEnableBatch,
					caCertId: caCertId,
					nbType: 'gNB'
				})).then(function(response){
					var data = response.data;

					if(data["success"]){
						vm.$message.success('<%=rb.getString("ZhengShuYiXiaFaQingShaoHouChaKanJieGuo")%>');
						vm.showConfirmInfoBatch = false;
						vm.$refs.gnbCertCtable.refresh();
					}else{
						vm.showConfirmInfoBatch = false;
						vm.$message.error(data["message"])
					}
					vm.certSelectData = [];
					vm.$refs.gnbCertCtable.clearSelection();
				})
	    	},
	    	//取消批量下发证书
	    	gnbCancelgnbIssueBatch(){
	    		var vm = this;

		    	vm.showConfirmInfoBatch = false;
		    	vm.selectBatchCaList = false;
		    	vm.selectBatchIssueFlag = false;
		    	vm.caFileIdBatch = '';
		    	vm.issueCertParamsBatch.searchText = '';
		    	vm.batchCertsSearchText = '';
		    	vm.certSelectData = [];
				vm.$refs.gnbCertCtable.clearSelection();
		    	vm.$refs['issueCertTableBatch'].clearSelection();
	    	},
	    	//文件导入搜索
	    	gnbIssueCertQueryBatch(){
	    		var vm = this;

	    		vm.issueCertParamsBatch.searchText = vm.batchCertsSearchText;
	    	},

	    	//-------------------------------------------------------------备份操作--------------------------------------------
	    	//批量备份
		    gnbBackupBatch(){
		    	var vm = this, certIds=[];

		    	if(vm.certSelectData.length == 0) return;

				certIds = vm.certSelectData.map(function(item){ return item.id});
		    	vm.$confirm('<%=rb.getString("QueDingPiLiangBeiFenZhengShu")%>', '<%=rb.getString("ZhengShuBeiFen")%>',{
					customClass: 'warningConfirm',
					confirmButtonText: '<%=rb.getString("QueDing")%>',
					cancelButtonText: '<%=rb.getString("QuXiao")%>',
				}).then(function(){
					axios.post('${ctx}/cell/cert/addBackupCertTask.action',stringify({
						id: certIds.join(','),
						nbType: 'gNB'
					})).then(function(response){
						var data = response.data;

						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.gnbCertCtable.refresh();//表格刷新
						}else{
							vm.$message.error(data["message"])
						}
						vm.certSelectData = [];
						vm.$refs.gnbCertCtable.clearSelection();//清空勾选的数据
					})
				}).catch(function(){})
		    },

	    	//-------------------------------------------------------------批量删除 下载 操作--------------------------------------------
            //批量下载
		    gnbDownloadBatch(){
		    	var vm = this, certIds=[];

				if(vm.certSelectData.length == 0) return;

				certIds = vm.certSelectData.map(function(item){ return item.id});

				exportByForm("${ctx}/cell/cert/downLoadIpsecPrivateCertFile.action",{
		        	id: certIds.join(','),
		        	nbType: 'gNB'
		        });
		    },
		 	//批量删除
			gnbDeleteBatch(){
				var vm = this, backupStatus = '', idArr = [];

				if(vm.certSelectData.length != 0){
					vm.certSelectData.map(function(item){
						if(item.backupUploadStatus !== '2'){
							idArr.push(item.id);
							backupStatus += item.backupUploadStatus + ",";
							return idArr;
						}
					})
					vm.batchDeleteIds = idArr;
					if(backupStatus == '' || backupStatus == null || backupStatus == undefined){
						vm.$message.info('<%=rb.getString("JinXingZhongDeRenWuBuKeYiShanChu")%>')
					}else{
						vm.gnbShowBatchDeleteInfo = true;
					}
				}else{
					return
				}
			},
			// 批量删除确认
			gnbBacthDeleteConfirm(){
				var vm = this;

				axios.post('${ctx}/cell/cert/deleteIpsecPrivateCertInfo.action', stringify({
					id: vm.batchDeleteIds.join(','),
					nbType: 'gNB'
				})).then(function(response){
					var data = response.data;

					if(data["success"]){
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.gnbCertCtable.refresh();//表格刷新
						vm.showCertInfoDialog = false;//详情关闭
					}else{
						vm.$message.error(data["message"])
					}
					vm.certSelectData = [];
					vm.$refs.gnbCertCtable.clearSelection();//清空勾选的数据
					vm.gnbShowBatchDeleteInfo = false;
				})

			},

	    	//-------------------------------------------------------------详情操作--------------------------------------------
			gnbOptInfoClick(row, ev){
				var vm = this;

				vm.showImportCard = false;
				vm.showSettingDialog = false;
				vm.showCertInfoDialog = true;
				axios.post('${ctx}/cell/cert/getIpsecCertInfoById.action',stringify({
					id : row.id,
					nbType: 'gNB',
					timeZone: timeZone
				})).then(function(response){
					var data = response.data;

					vm.infoSerialNumber = data.serialNumber;
					vm.caUploader = data.caUploader;
					vm.caUploadTime = data.caUploadTime;
					vm.caDownTime = data.caDownTime;//ca 证书 下发时间
					vm.caDescription = data.caDescription;
					vm.privateUploader = data.privateUploader;
					vm.privateUploadTime = data.privateUploadTime;
					vm.privateDownTime = data.privateDownTime;
					vm.privateDescription = data.privateDescription;

					//0 == 下发失败，1 == 已下发，2 == 下发中，3 == 未下发 -->
					//如果证书为空时
					if(data.caFileName == '' || data.caFileName == null || data.caFileName == undefined){
						vm.caStatus = '';
					}else{
						if(data.caUploadStatus == '0'){
							vm.caFileName = '<i class="el-icon el-icon-status-issued issueErrorColor issueInfo"></i>'+data.caFileName;
						}else if(data.caUploadStatus == '1'){
							vm.caFileName = '<i class="el-icon el-icon-status-issued issueOkColor issueInfo"></i>'+data.caFileName;
						}else if(data.caUploadStatus == '2'){
							vm.caFileName = '<i class="status_issuing issueInfo"></i>'+data.caFileName;
						}else if(data.caUploadStatus == '3'){
							vm.caFileName = '<i class="el-icon el-icon-status-issued issueNoColor issueInfo"></i>'+data.caFileName;
						}
					}
					//ipsec 证书
					if(data.certFileName == '' || data.certFileName == null || data.certFileName == undefined){
						vm.certFileName = '';
					}else{
						if(data.certUploadStatus == '0'){
							vm.certFileName = '<i class="el-icon el-icon-status-issued issueErrorColor issueInfo"></i>'+data.certFileName;
						}else if(data.certUploadStatus == '1'){
							vm.certFileName = '<i class="el-icon el-icon-status-issued issueOkColor issueInfo"></i>'+data.certFileName;
						}else if(data.certUploadStatus == '2'){
							vm.certFileName = '<i class="status_issuing issueInfo"  style="margin-top: 3px;"></i>'+data.certFileName;
						}else if(data.certUploadStatus == '3'){
							vm.certFileName = '<i class="el-icon el-icon-status-issued issueNoColor issueInpro"></i>'+data.certFileName;
						}
					}

					//秘钥 证书
					if(data.secretKeyFileName == '' || data.secretKeyFileName == null || data.secretKeyFileName == undefined){
						vm.secretKeyFileName = '';
					}else{

						if(data.secretKeyUploadStatus == '0'){
							vm.secretKeyFileName = '<i class="el-icon el-icon-status-issued issueErrorColor issueInfo"></i>'+data.secretKeyFileName;
						}else if(data.secretKeyUploadStatus == '1'){
							vm.secretKeyFileName = '<i class="el-icon el-icon-status-issued issueOkColor issueInfo"></i>'+data.secretKeyFileName;
						}else if(data.secretKeyUploadStatus == '2'){
							vm.secretKeyFileName = '<i class="status_issuing issueInfo" style="margin-top: 3px;"></i>'+data.secretKeyFileName;
						}else if(data.secretKeyUploadStatus == '3'){
							vm.secretKeyFileName = '<i class="el-icon el-icon-status-issued issueNoColor issueInpro"></i>'+data.secretKeyFileName;
						}
					}
				})
			},
            //关闭证书详情窗口
            gnbCloseCertInfoDialog(){
				var vm = this;

	    		vm.showCertInfoDialog = false;
			},
			//-------------------------------------------------------------跳转 CA 证书--------------------------------------------
			//跳转CA证书页面
			gnbCACertificate(){
				var vm = this;

	        	vm.slideHeader = false;
	        	vm.slideFooter = false;
	    	    vm.slideUrl = "${ctx}/cell/cert/toGNBCACertificate.action",
	    	    vm.slidePosition = 'left'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
    	    	vm.$refs.slideCA.showSlide();

	    	    vm.serialNumParams.searchText = '';
				vm.snSearchText = '';
				vm.certsName = '';
				vm.secretKeyName = '';
	    	    vm.certSelectData = [];
				vm.$refs.gnbCertCtable.clearSelection();//清空勾选的数据
				vm.$refs.ruleForm.resetFields();
				vm.showImportCard = false;
				vm.showSettingDialog = false;
				vm.showCertInfoDialog = false;
			},
			//关闭跳转 CA 证书页面
			gnbCancelSlide(){
	    		var vm = this;

	    		vm.$refs.slideCA.hide();
	    	},

			//-----------------------------------------------------------------已选-----------------------------------------------
	    	// 打开已选弹窗
            gnbOpenBulkSelectTable(){
                var vm = this;

                vm.bulkSelectShow = true
            },
            // 关闭已选弹窗
            gnbCloseBulkSelectTable(){
                var vm = this;

                vm.bulkSelectShow = false;
            },
            // 设备已选表格 清空事件
            gnbClearBulkSelected(){
                var vm = this;
				if(vm.gnbActiveName == 'gnbIpsecCert'){
					vm.$refs["gnbCertCtable"].clearSelection();
				}else if(vm.gnbActiveName == 'gnbSslCert'){
					vm.$refs["gnbSslCertTable"].clearSelection();
				}
                
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            gnbDelBulkSelected(rows){
				var vm = this, 
					activeName = vm.gnbActiveName,
					rowKey = 'serialNumber',
					tabsCodes = {
						'gnbIpsecCert':'gnbCertCtable',
						'gnbSslCert':'gnbSslCertTable'
					},
					tabs = tabsCodes[activeName];

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
            },

			//-------------------------------------------------------------SSL 证书 -------------------------------------------------------------
			/**
			*  列表选中
			* @param selection{Array}   选中数据
			*/
			gnbSslCertBatchSelect(selection){
				var vm = this; 
				
			    vm.certSelectData = selection.map((item)=>{
					return Object.assign(item,{snNumber: item.serialNumber})
				}); 
			},
			// 给全部设备下发证书
	    	gnbAllDeviceIssuedCertificate(){
	    		var vm = this;
	    		
	    		vm.sendSSLCertFlag = 'all';
	    		
	    		vm.gnbCommonSSLCertInfoList();
	    		vm.gnbShowSendSSLCertificate = true;
	    	},
	    	//单个或批量下发证书
	    	gnbSingleDeviceIssuedCertificate(){
				var vm = this;
				
				vm.sendSSLCertFlag = 'single';				
				if(vm.certSelectData.length == 0) return;
	    		vm.gnbCommonSSLCertInfoList();
	    		vm.gnbShowSendSSLCertificate = true;
	    	},
	    	//获取 SSL 证书下拉菜单
	    	gnbCommonSSLCertInfoList(){
	    		var vm = this;
	    		
	    		axios.post("${ctx}/cert/ssl/getSelectGNBSSLCertInfoList.action").then(function(response){
					var data = response.data;
					
					if(data && data.length > 0){
						vm.gnbSslCertList = data;
					}
				})
	    	},
	    	//确认下发 SSL 证书
	    	gnbConfirmSendCertificate(){
	    		var vm = this, params = {}, snList = vm.certSelectData.map(function(item){ return item.serialNumber});
	    		
	    		if(vm.sendSSLCertFlag == 'single'){	    			
	    			params.serialNumbers = snList.join(','); 
	    			params.fileId = vm.gnbSendSSLCertForm.sslCert;
	    			params.executeType = 'SELECT';
	    		}else{
	    			params.fileId = vm.gnbSendSSLCertForm.sslCert;
	    			params.executeType = 'ALL';
	    		}
	    		
				vm.$refs.gnbSendSSLCertForm.validate((valid) => {
                   if (valid) {
                	   axios.post('${ctx}/cert/ssl/sendGNBSSLCert.action',stringify(params)).then(function(response){
							var data = response.data;
							
							if(data) {
								if(data["success"]){									
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>',
										type: 'success'
									});
									vm.$refs.gnbSslCertTable.refresh();
									vm.gnbShowSendSSLCertificate = false;
								}else{
									vm.$message.error(data["message"])
								}
								//置空已选
								vm.$refs["gnbSslCertTable"].clearSelection();
							}
						}).catch(function(error){})                    
                   }
               })
	    	},
	    	//取消下发 SSL 证书
	    	gnbCancelSendCertificate(){
				var vm = this;
	    		
				vm.$refs.gnbSendSSLCertForm.resetFields();
	    		vm.gnbShowSendSSLCertificate = false;
	    	},
	    	//跳转到 SSL 证书维护页面
	    	gnbJumpSslCertificatePage(){
				var vm = this;
				
				vm.slideHeader = false
				//vm.slideUrl = "${ctx}/cert/ssl/toSSLCertFileManage.action"
	    	   	vm.slideUrl = "${ctx}/cert/ssl/toGNBSSLCertManage.action"
	    	    vm.slideFooter = false
	    	    vm.slidePosition = 'top'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
	    	    vm.$refs.gnbSlideCertManagePage.showSlide(function(){});
	    	},
	    	//ssl 证书 查询
	    	gnbQuerySSLCertificate(val){
	    		var vm = this;
	    		
	    		Object.assign(vm.gnbSslCertificateParams,{	    			
	    			searchText: val	    		
	    		}) 
	    	},
	    	//关闭一级slide，跳转到 omc 证书管理页面或者跳转到ssl 证书维护页面
			gnbSslcancelSlide(){	    		
	    		var vm = this;
	    		
	    		vm.$refs.gnbSlideCertManagePage.hide();
	    	},
	    },
	    mounted(){
	    	this.gnbInit();
		}
	});
</script>
