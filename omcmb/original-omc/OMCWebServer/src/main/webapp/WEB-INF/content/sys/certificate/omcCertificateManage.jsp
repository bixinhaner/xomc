<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
	#certificateManagePage{
		height: 100%;
		display: flex;
		width:100%;
	}	
	#certificateManagePage .el-card__body {
		background: #fff;
		border: none;
		/*padding: 20px;*/
	}	
	#certificateManagePage .el-card__footer {
		display: none;
	}	
	#certificateManagePage .el-input__suffix {
		top: 4px;
	}
	#certificateManagePage .el-textarea__inner {
		line-height: 1.2;
		padding: 2px 15px;
		resize: none;
	}	
	#certificateManagePage .el-upload__tip {
		margin-top: 0;
	}
	#certificateManagePage .slide-content{
		margin-left:0 !important;
	}
	#certificateManagePage .el-dialog__body{
		padding:24px 20px 20px; 
		min-height:98px;	
	}
	#certificateManagePage .el-dialog__footer{
		padding:20px 20px 26px;
		text-align:left;
	}
	#certificateManagePage .el-checkbox-group{
		padding:16px 0 10px;
	}
	#certificateManagePage .w320{
		width:320px;
	}
	#certificateManagePage .issueStatus{
		margin-top:4px;
		font-size:17px; 
		margin-right:10px;
		float:left;
	}
	#certificateManagePage .issueNoColor:before{		
		color:#CFCFCF;
	}
	#certificateManagePage .issueOkColor:before{
		color:#67D972;
	}
	#certificateManagePage .statusTip{
		overflow:hidden;
		text-overflow:ellipsis;
		white-space:nowrap;
		font-size:12px !important;
	}
	#certificateManagePage .el-checkbox__input.is-checked+.el-checkbox__label{
		color:#333;
		font-weight:normal;
	}
	#certificateManagePage .caCertList .el-form-item:nth-child(2) .el-form-item__label{
		font-size:12px;
		padding-bottom:4px;
	}
	#certificateManagePage .snTableBorder{
		border:1px solid #DFE2EE;
	}

	#certificateManagePage .issueErrorColor:before,
	#certificateManagePage .statusError:before{
		color:#E88282;
	}

	#certificateManagePage .tableBackup{
		font-size:12px;
		color:#4D84FF;
		cursor:pointer;
		text-decoration:underline;
	}
	#certificateManagePage .backupUpload{
		margin-left:5px;
		float:left;
		width:81%;
		overflow:hidden;
		text-overflow:ellipsis;
		white-space:nowrap;
	}
	/* 最新样式表 */
	#certificateManagePage .el-form-item{
		margin-bottom:26px !important;
	}
	
	#certificateManagePage .el-form-item__content{
		line-height:26px;
	}
	#certificateManagePage .taskInput{
		width:172px;
	}
	#certificateManagePage .el-message-box{
		padding-bottom:20px;
	}
	#certificateManagePage .el-message-box__btns > .el-button:not(:first-child){
   		 margin: 0 15px 0 0;
	}
	#certificateManagePage .el-message-box__content{
		padding:30px;
	}
	#certificateManagePage .el-message-box__message p {
		margin:0 !important;
	}
	/* 确定按钮样式 */
	/*#certificateManagePage .warningConfirmTip .el-button--primary{
		background-color:#4D84FF !important;
		color:#fff !important;
		border:1px solid #4D84FF !important;
	}
	#certificateManagePage .warningConfirmTip .el-button--primary:hover{
		background-color:#94B5FF !important;
		color:#fff !important;
		border:1px solid #94B5FF !important;
	}
	#certificateManagePage .warningConfirmTip .el-button--primary:active{
		background-color:#2A61DB !important;
		color:#fff !important;
		border:1px solid #2A61DB !important;
	}	
	#certificateManagePage .warningConfirmTip .el-button{
		background-color:#FFF;
		color:#666;
		border:1px solid #DCDFE6;
	}
	#certificateManagePage .warningConfirmTip .el-button:hover{
		background-color:#F2F9FF;
		color:#4D84FF;
		border:1px solid #4D84FF;
	}
	#certificateManagePage .warningConfirmTip .el-button:active{
		background-color:#F2F9FF;
		color:#4D84FF;
		border:1px solid #2A61DB;
	}*/
	#certificateManagePage .el-tooltip__popper{
		padding:10px;
	}
	#certificateManagePage .statusProgress{
		font-size:12px !important;
	}
	#certificateManagePage .batchDeleteWarp .el-dialog__body{
		min-height:60px !important;
	}
	#certificateManagePage .el-input-group__append {
		padding: 0 6px;
	}
	#certificateManagePage .el-input-group__append .el-icon:before { color: #7A7992; }
	
	#certificateManagePage .issueInfo{
		margin-right:8px;
		font-size:18px;
	}
	#certificateManagePage .issueInpro{
		margin-right:8px;
		font-size:18px;
	}
	#certificateManagePage .commonQuery .pairgrid-query {
		width: 400px !important;
	}
</style>
<!--设备证书页面  -->
<div class="overflow-cls">
	<div id="certificateManagePage" class='commonFlex' style="min-width: 900px;position: relative;overflow-y: hidden;">
		<div class='leftWarp commonWarp'>
			<div class="circleIcon placeholder-bt" style="right:86px;top:42px" placeholder="<%=rb.getString("CAZhengShu")%>">		
				<span class="el-icon el-icon-circle-certificate" @click="CACertificate"></span>
			</div>
			<div v-if="hasLicenseRole" class="circleIcon placeholder-bt" style="right: 46px;top:42px"placeholder="<%=rb.getString("ZhengShuDaoRu")%>">		
				<span class="el-icon el-icon-circle-import" @click="importCertificate"></span>
			</div>
			<div class="circleIcon placeholder-bt" style="right:10px;top:42px" placeholder="<%=rb.getString("GuanBi")%>">
				<span class="el-icon el-icon-circle-close" @click="closeCertManageSlider"></span>
			</div>
			<div class='leftBoxHeader'><%=rb.getString("OMCZhengShuGuanLi")%></div>
			<!--主体内容-->
			<el-ctable ref="ctable" :time="6" :url="certManageUrl" :query-params="certManageParams" id="certManageTable" :limit="limitBatch"
           		:page-size="pageSize" pagination="true" :rownumber=true :row-key="'serialNumber'" @selection-change='batchSelect'>
				<template slot="toolbar">
               		<div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
                     	<!-- 已选数据 -->
						<div class="selectBlukBoxCls">
							<div class="selectMain headBtnItemCls">
                             	<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable" style='border-right: 0; padding: 0;'>
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
                     	<!-- 当同时执行下发证书、备份证书、更新证书时，则按照时间顺序执行，第一个任务执行结束后执行第二个任务； -->
						<div v-if="hasLicenseRole" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="issueBatch">
                    		<span class='el-icon el-icon-status-issued'></span>
                    		<span><%=rb.getString("XiaFaZhengShu")%></span>
		                </div>
		                <div v-if="hasLicenseRole" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="backupBatch">
		                     <span class='el-icon el-icon-operation-backups'></span>
		                     <span><%=rb.getString("ZhengShuBeiFen")%></span>
                    	</div>
                    	<div :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="downloadBatch">
                    		<span class='el-icon el-icon-operation-download'></span>
                    		<span><%=rb.getString("PiLiangXiaZai")%></span>
                    	</div>
                    	<div v-if="hasLicenseRole" :class="certSelectData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="deleteBatch" style='border-right: 0;'>
                    		<span class='el-icon el-icon-operation-delete'></span>
                    		<span><%=rb.getString("PiLiangShanChu")%></span>
                    	</div>
                    </div>
                    <div class='toolbarHeadBtnBoxCls commonQuery tableHeadQueryBoxCls' style=' border: 0; padding: 5px 0;'>
                    	<el-query type="normal" @query="queryIpsec" placeholder="<%=rb.getString("ZhengShuSouSuo")%>"></el-query>
                    </div>
                </template>
                <el-table-column type="selection" :reserve-selection="true"></el-table-column>
				<el-table-column v-if="hasLicenseRole" width="40">
					<template slot-scope="scope">
                        <div class="el-icon el-icon-operation-info" @click="optInfoClick(scope.row,event)"></div>
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
								<a class="tableBackup" @click="backupDownload(scope.row.id)"> {{scope.row.backupFileName}}</a>
							</div>
						</div>
						<div v-if="scope.row.backupUploadStatus == '2'" class="statusProgress">						
							<span class="status_backup" style="float:left;"></span>
							<span class="backupUpload"><%=rb.getString("ZSBeiFenZhong")%></span>
						</div>						
					</template> 
                </el-table-column> 
			</el-ctable>           
		</div>
	
		<!-- 导入文件框 -->
		<div class='rightWarp' style='position: relative; flex: 0 1 360px;' v-show='showImportCard'>
			<div class='rightWarpLayer'>
		 		<div class='rightBoxHeaderHasTip'>
					<div class='headerText'>
						<span><%=rb.getString("ZhengShuDaoRu")%> <%=rb.getString("ipsecCert")%></span>
						<span class='closeIconBox' @click='closeImportDevice'><i class='el-icon el-icon-close'></i></span>
					</div>
				</div>
				<div class='rightWarpLayerContent'>
					<el-form label-position="top" ref="ruleForm" :model='ruleForm' :rules='rules' style='padding: 30px 20px;'>     					      
		                <el-form-item label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serialNumber'>
							<el-input maxlength="50" v-model="ruleForm.serialNumber" class="taskInput" @change="inputChange" style='width: 320px;'>
								<div slot="append">
									<span class="el-icon el-icon-common-select commonTemplateText12" @click="selectDevice"> <%=rb.getString("XuanZeYiYouSheBei")%></span>
								</div>
							</el-input>
						</el-form-item>
						<el-form-item v-show="snTableSelect">
		                    <el-ctable ref="enbImportDevices" id='enbImportDevices' :url="snTableUrl" :time='6' :query-params="serialNumParams" :row-key="'serialNumber'" style='border-radius: 4px; border: 1px solid #DFE2EE; margin: 0;'
				            	:show-pager="false" :page-size="pageSize" pagination="false" :rownumber=false height="330px"
				            	@row-click="snChange">
				               <template slot="toolbar">
				               		<div class="searchCon" style="margin-left:18px;">
										<el-input placeholder="<%=rb.getString("XiaoZhanBianMa")%>" suffic-icon='el-icon-search' v-model="snSearchText" style="width:270px;">
											<i slot="suffix" class="el-icon el-icon-common-search"  @click="querySnSearch"></i>
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
	                        <el-upload :on-success='checkFile' :on-change="fileChange" :show-file-list=false ref="uploadCert" 
	                        	:action="ruleForm.uploadFileUrl" :data="fileParams" name="uploadCertFile" :auto-upload="false">
	                            <el-input :readonly="true" :value=certsName class="w320">
									<a slot="append" class="el-icon el-icon-operation-import" @click="fileSelect"></a>
								</el-input>	                            
	                            <a slot="trigger" ref="file_up"></a>	                            
	                        </el-upload>
                    	</el-form-item>
                    	<el-form-item label="<%=rb.getString("MiYaoZhengShu")%>" prop='secretKeyName'>
	                        <el-col :span="10">
	                            <el-upload :on-success='checkFile2' :on-change="fileChangePrivateKey" :show-file-list=false ref="uploadCert" 
	                            	:action="ruleForm.uploadFileUrl" name="uploadSecretKeyFile"
	                                :data="fileParams" :auto-upload="false">
	                                <el-input :readonly="true" :value=secretKeyName class="w320">
	                                    <a slot="append" class="el-icon el-icon-operation-import" @click="fileSelectPrivateKey"></a>
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
						<el-button type="primary" @click="uploadDevice"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="closeImportDevice"><%=rb.getString("QuXiao")%></el-button>
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
						<span class='closeIconBox' @click=closeCertInfoDialog><i class='el-icon el-icon-close'></i></span>
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
			:close-on-click-modal="false" :modal-append-to-body="false" @close="cancelIssueBatch">
			<el-form :model="confirmFormBatch" ref="confirmFormBatch" label-position="top">
				<el-form-item label="<%=rb.getString("QueDingPiLiangXiaFaCiZhengShuMa")%>" >
					<el-checkbox v-model="caCertEnableBatch" checked label="<%=rb.getString("CAZhengShu")%>"></el-checkbox> 
					<el-checkbox v-model="deviceCertEnableBatch" checked label="<%=rb.getString("SheBeiZhengShuIpsecZhengshuMiYaoZhengShu")%>"></el-checkbox>
					<div slot="tip" class="el-upload__tip" v-show="selectBatchIssueFlag"><%=rb.getString("CAZhengShuSheBeiZhengShuBiXuanQiYi")%></div>
				</el-form-item> 			
				<el-form-item label="<%=rb.getString("CAZhengShu")%>" v-show="caTableSelectBatch" style="margin-bottom:0px !important;">
					<el-ctable ref="issueCertTableBatch" :url="batchIssueCaCertUrl" :query-params="issueCertParamsBatch" class="snTableBorder"				
		            	:page-size="pageSize" pagination="true" :rownumber=false :row-key="'id'" height="330px" 
		            	@row-click="issueChangeBatch">
		                <template slot="toolbar">
		                	<div class="searchCon" style="margin-left:18px;">
								<el-input placeholder="<%=rb.getString("ZhengShuWenJian")%>" suffic-icon='el-icon-search' v-model="batchCertsSearchText" style="width:270px;">
									<i slot="suffix" class="el-icon el-icon-common-search"  @click="issueCertQueryBatch"></i>
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
					<el-button type="primary" @click="confirmIssueBacth"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="cancelIssueBatch"><%=rb.getString("QuXiao")%></el-button>
				</div>	
			</div>
		</el-dialog>	
		
		<el-dialog title="<%=rb.getString("QueRen")%>" :visible="showBatchDeleteInfo" width="420" class="batchDeleteWarp"
			:close-on-click-modal="false" :modal-append-to-body="false" @close="showBatchDeleteInfo = false">
			<div><%=rb.getString("QueDingPiLiangShanChuZhengShu")%></div>
			<div style="padding-top:10px;"><%=rb.getString("JinXingZhongDeRenWuBuKeYiShanChu")%></div>
			<span slot="footer" class="dialog-footer">
				<div style="text-align:right;">
					<el-button type="primary" @click="bacthDeleteConfirm"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="showBatchDeleteInfo = false"><%=rb.getString("QuXiao")%></el-button>
				</div>	
			</span>
		</el-dialog>
		
		<!-- 跳转到CA 证书管理页面 -->		
		<el-slide ref="slideCA" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
			:width="slideWidth" @cancel='cancelSlide' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>	
	</div>
</div>
<script type="text/javascript">
	if(window.certificateManageVue) {
		try {
			window.certificateManageVue.$destroy();
		}catch(e){}
	}
	window.certificateManageVue = new Vue({
	    el: '#certificateManagePage',
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
				};
	    	return {
				showBatchDeleteInfo: false,
	    		pageSize: 50,
	    		certManageParams: {
	    			timeZone: timeZone,
	             	searchText: '',
	             	nbType: 'eNB'
	            },
	            snSearchText: '',
	            serialNumParams: {
	            	timeZone: timeZone,
	             	searchText: ''
	            },

	            batchCertsSearchText: '',
	            issueCertParamsBatch: {
	            	timeZone: timeZone,
	             	searchText: '',
	             	nbType: 'eNB'
	            },
	            certManageUrl: '',
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
	           	rules: {
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
	    	neType() {
				return sysMain.headType;
			},
			limitBatch(){
				return batchOperation ? '' : 1;
			},
			hasLicenseRole() {
				return  writableMap.CODE_IPSEC_CERT == true;
			}
	    },
	    watch: {
	    	neType() {
	    		var vm = this;
	    		
	    		vm.init();
			},
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
	    	init(){
	    		var vm = this;

	    		vm.certManageUrl = "${ctx}/cell/cert/getIpsecCertInfoPageList.action";
	    	},
	    	
	    	/**
			* 列表选中
			* @param selection{Array}   选中数据
			*/
			batchSelect(selection){
				var vm = this; 
				
			    vm.certSelectData = selection.map((item)=>{
					return Object.assign(item,{snNumber:item.serialNumber})
				}); 
			},
			
			//表格操作-备份点击下载文件
			backupDownload(id){	
				var vm = this;
				
		    	exportByForm("${ctx}/cell/cert/downLoadIpsecCertBackupFile.action",{		        	
		        	id: id,
		        	nbType: 'eNB'
		        });
			},
	    	//Ipsec 模块  搜索事件 
	    	queryIpsec(val){
				var vm = this;
				
	    		Object.assign(vm.certManageParams,{	    			
	    			searchText: val	    		
	    		}) 
	    	},
	    	
	    	//-------------------------------------------------------------ipsec 证书 导入--------------------------------------------
			//导入设备证书点击事件			 
			importCertificate(){
				var vm = this;
				
				vm.serialNumber = '';
				vm.serialNumberRadio = '';
				vm.$refs.ruleForm.resetFields();
				vm.$refs.ctable.clearSelection();//清空勾选的数据	   
				vm.showImportCard = true;
				vm.snTableSelect = false;
				vm.showSettingDialog = false;
				vm.showCertInfoDialog = false;
			},
	    	//点击选择已有设备
	    	selectDevice(){
				var vm = this;

				vm.$refs.enbImportDevices.refresh();
	    		vm.snTableSelect = true;
	    	},
	    	//sn选择已有设备
	    	snChange(row,old){
	    		var vm = this;
	    		
				if(row) {
					vm.serialNumberRadio = row.serialNumber;
					vm.ruleForm.serialNumber = row.serialNumber;	
				}
	    	},
	    	//导入-输入的基站编码input 改变时
	    	inputChange(value,newVal){
	    		var vm = this;
	    		
	    		vm.snTableSelect = false;
	    		vm.serialNumberRadio = '';
	    		vm.$refs.enbImportDevices.clearSelection();
	    	},
	    	
	    	//-----------------------------------ipsec 证书选择-----------------------------------
	    	checkFile(res, file) {
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
                var fileList = vm.$refs.uploadCert.uploadFiles;
                fileList.forEach(function (file) {
                    file.status = 'ready';
                })
            },
            /**
           	  *选择文件后，校验格式，并赋值页面显示 
             *@param file：文件名称
             */
            fileChange(file, fileList) {
                var vm = this;         
                
                vm.fileParams.uploadCertFile = file.raw;
                vm.certsName = file.name;
            },
         	// 导入文件按钮
            fileSelect() { 
                var vm = this;
            
                vm.$refs.uploadCert.clearFiles();
                vm.$refs['file_up'].click();
            },
            
	    	//-----------------------------------秘钥证书-----------------------------------
	    	//发送请求，校验device文件内容 
	    	checkFile2(res, file) {    
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
                var fileList = vm.$refs.uploadCert.uploadFiles;
                fileList.forEach(function (file) {
                    file.status = 'ready';
                })
            },
            /**
             * PrivateKey 导入监听
             * file:文件
             *fileList：文件列表
             */
             fileChangePrivateKey(file, fileList) {
                 var vm = this;
                
                 vm.fileParams.uploadSecretKeyFile = file.raw;
                 vm.secretKeyName = file.name;
             },        
             fileSelectPrivateKey() {  //PrivateKey 导入文件按钮
                 var vm = this;
             
                 vm.$refs.uploadCert.clearFiles();
                 vm.$refs['file_up2'].click();
             },
             /*确定导入*/
             uploadDevice() {
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
                 vm.fileParams.nbType = 'eNB';
                 vm.$refs.ruleForm.validate((valid) => {
                     if (valid && secretKeyName && certsName) {
                         this.uploadFiles(vm.ruleForm.uploadFileUrl, vm.fileParams)                      
                     }
                 })
			},                                        
            /*
           	* 导入函数
            * url：当前的修改或者添加url
            * params： 所有的from参数
            */
            uploadFiles(url, params, cb) {
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
                            vm.$refs.ctable.refresh();
                            vm.showImportCard = false; 		
            				vm.certsName = '';
            				vm.secretKeyName = '';
            				vm.serialNumber = '';
            				vm.serialNumberRadio = '';          				
            				vm.snTableSelect = false;
            				vm.$refs.enbImportDevices.clearSelection();
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
			closeImportDevice(){
				var vm = this;
				
				vm.showImportCard = false;			
				vm.certsName = '';
				vm.secretKeyName = '';
				vm.serialNumber = '';
				vm.serialNumberRadio = '';
				vm.snTableSelect = false;
				vm.serialNumParams.searchText = '';
				vm.snSearchText = '';
				vm.$refs.enbImportDevices.clearSelection();//清空表格勾选的状态
				vm.$refs.ruleForm.resetFields();
			},
			//导入基站编码查询
	    	querySnSearch(){
	    		var vm = this;
	    		
				vm.serialNumParams.searchText = vm.snSearchText;
	    	},

	    	//-------------------------------------------------------------下发操作--------------------------------------------
		    //选择设备选中事件
	    	issueChangeBatch(row, old){	    		
				var vm = this;
				
				if(row) {
					vm.caFileIdBatch = row.id;	
					vm.selectBatchCaList = false;
				}		
			},
	    	//批量下发证书
		    issueBatch(){
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
	    	confirmIssueBacth(){
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
					nbType: 'eNB'
				})).then(function(response){
					var data = response.data;
					
					if(data["success"]){
						vm.$message.success('<%=rb.getString("ZhengShuYiXiaFaQingShaoHouChaKanJieGuo")%>');
						vm.showConfirmInfoBatch = false;
						vm.$refs.ctable.refresh();				
					}else{
						vm.showConfirmInfoBatch = false;						
						vm.$message.error(data["message"])
					}
					vm.certSelectData = [];
					vm.$refs.ctable.clearSelection();
				}) 	    		
	    	},
	    	//取消批量下发证书
	    	cancelIssueBatch(){
	    		var vm = this;
	    		
		    	vm.showConfirmInfoBatch = false;	
		    	vm.selectBatchCaList = false;
		    	vm.selectBatchIssueFlag = false;
		    	vm.caFileIdBatch = '';
		    	vm.issueCertParamsBatch.searchText = '';
		    	vm.batchCertsSearchText = '';
		    	vm.certSelectData = [];
				vm.$refs.ctable.clearSelection();
		    	vm.$refs['issueCertTableBatch'].clearSelection();
	    	},
	    	//文件导入搜索
	    	issueCertQueryBatch(){
	    		var vm = this;
	    		
	    		vm.issueCertParamsBatch.searchText = vm.batchCertsSearchText;
	    	},
	    	
	    	//-------------------------------------------------------------备份操作--------------------------------------------	
	    	//批量备份
		    backupBatch(){
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
						nbType: 'eNB'
					})).then(function(response){
						var data = response.data;
						
						if(data["success"]){
							vm.$message.success('<%=rb.getString("ChengGong")%>');
							vm.$refs.ctable.refresh();//表格刷新							
						}else{
							vm.$message.error(data["message"])
						}
						vm.certSelectData = [];
						vm.$refs.ctable.clearSelection();//清空勾选的数据
					})
				}).catch(function(){})
		    },
		    
	    	//-------------------------------------------------------------批量删除 下载 操作--------------------------------------------
            //批量下载
		    downloadBatch(){
		    	var vm = this, certIds=[];
				
				if(vm.certSelectData.length == 0) return;

				certIds = vm.certSelectData.map(function(item){ return item.id});

				exportByForm("${ctx}/cell/cert/downLoadIpsecPrivateCertFile.action",{		        	
		        	id: certIds.join(','),
		        	nbType: 'eNB'
		        });
		    },
		 	//批量删除
			deleteBatch(){
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
						vm.showBatchDeleteInfo = true;										
					}
				}else{
					return
				}								
			},
			// 批量删除确认
			bacthDeleteConfirm(){
				var vm = this;
				
				axios.post('${ctx}/cell/cert/deleteIpsecPrivateCertInfo.action', stringify({
					id: vm.batchDeleteIds.join(','),
					nbType: 'eNB'
				})).then(function(response){
					var data = response.data;
					
					if(data["success"]){
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.ctable.refresh();//表格刷新
						vm.showCertInfoDialog = false; //数据删除时，已打开的详情页也需关闭
					}else{
						vm.$message.error(data["message"])
					}
					vm.certSelectData = [];
					vm.$refs.ctable.clearSelection();//清空勾选的数据
					vm.showBatchDeleteInfo = false;	
				})
				
			},
			
	    	//-------------------------------------------------------------详情操作--------------------------------------------
			optInfoClick(row, ev){
				var vm = this;
				
				vm.showImportCard = false; 
				vm.showSettingDialog = false;
				vm.showCertInfoDialog = true;
				axios.post('${ctx}/cell/cert/getIpsecCertInfoById.action',stringify({
					id : row.id,
					nbType: 'eNB',
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
            closeCertInfoDialog(){
				var vm = this;
				
	    		vm.showCertInfoDialog = false;
			},
			//-------------------------------------------------------------跳转 CA 证书--------------------------------------------		
			//跳转CA证书页面			
			CACertificate(){
				var vm = this;
				
	        	vm.slideHeader = false;
	        	vm.slideFooter = false;
	    	    vm.slideUrl = "${ctx}/cell/cert/toCACertificate.action",
	    	    vm.slidePosition = 'left'
	    	    vm.slideHeight = '100%'
	    	    vm.slideWidth = '100%'
    	    	vm.$refs.slideCA.showSlide();
	    	     
	    	    vm.serialNumParams.searchText = '';
				vm.snSearchText = '';
				vm.certsName = '';
				vm.secretKeyName = '';
	    	    vm.certSelectData = [];
				vm.$refs.ctable.clearSelection();//清空勾选的数据
				vm.$refs.ruleForm.resetFields();
				vm.showImportCard = false; 
				vm.showSettingDialog = false;
				vm.showCertInfoDialog = false;
			},
			//关闭跳转 CA 证书页面
			cancelSlide(){	    		
	    		var vm = this;
	    		
	    		vm.$refs.slideCA.hide();
	    	},
	    	
			//-----------------------------------------------------------------已选-----------------------------------------------
	    	// 打开已选弹窗
            openBulkSelectTable(){
                var vm = this;
                
                vm.bulkSelectShow = true
            },
            // 关闭已选弹窗
            closeBulkSelectTable(){
                var vm = this;
                
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 清空事件
            clearBulkSelected(){
                var vm = this;
				
                vm.$refs["ctable"].clearSelection();
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            delBulkSelected(rows){
				var vm = this,
					tabs = 'ctable',
					rowKey = 'serialNumber';
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
			//关闭 omc 证书管理页面
			closeCertManageSlider(){
				certificateVue.$refs.slideCertManagePage.hide();
			}
	    },
	    mounted(){
	    	this.init();
		}
	});
</script>
