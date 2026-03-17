<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addRuleDiv .el-collapse-item__header{
	border-bottom:1px solid #fff;
}
#addRuleDiv .el-collapse-item__arrow{
	position:absolute;
	left:23px;
	top:0px;
}
#addRuleDiv .el-collapse-item{
	position:relative;
}
#addRuleDiv .el-icon-arrow-right{
	font-size:16px;
}
#addRuleDiv .el-collapse-item__header .el-icon-arrow-right:before{
	content:"\e639";
	color:#BBB;
}
#addRuleDiv .el-collapse-item__header .is-active.el-icon-arrow-right:before{
	content:"\e638";
	color:#BBB;
}
#addRuleDiv .el-collapse-item__arrow.is-active{
	transform:rotate(0deg);
}
#addRuleDiv .el-collapse{
	border-top:1px solid #fff;
	border-bottom:1px solid #fff;
}
#addRuleDiv .el-collapse-item__wrap{
	border-bottom:1px solid #fff;
}
#addRuleDiv .el-icon-circle-info:before,.dialogStyle .el-icon-circle-info:before{
	color:#333333;
}
#addRuleDiv .btn-next .el-icon-arrow-right{
	font-size:12px;
}
#addRuleDiv .btn-next .el-icon-arrow-right:before{
	content:"\e794"
}
#addRuleDiv .dialogStyle .el-input{
	width:585px;
}
#addRuleDiv .resultDialog p{
	font-size:12px;
	margin-bottom:5px;
}
#addRuleDiv .suffixItem{
	display:inline-block;
	margin-bottom:0px;
	margin-right:5px;
}
#addRuleDiv .suffixItem .el-form-item__content{
	line-height:16px;
}
#addRuleDiv .form-suffix .text{
	overflow:hidden;
	white-space:nowrap;
	text-overflow:ellipsis;
}
#addRuleDiv .typeFormItem .el-form-item__content{
	line-height:48px;
}
.accessControlDialogCls .gpsItem .form-suffix{
	width:auto;
}
.accessControlDialogCls .gpsItem .form-suffix .text{
	width:auto;
}
#addRuleDiv .el-icon-circle-info:before,.dialogStyle .el-icon-circle-info:before{
	color:#CFCFCF;
}
#addRuleDiv .el-form-item__label{
	line-height:28px;
}
#addRuleDiv .typeItem{
	height:350px;
	border:1px solid #DEDFE6;
	border-top:1px solid #4D84FF;
	border-radius:4px;
}
#addRuleDiv .el-textarea{
	width:400px;
}
#addRuleDiv .queryGroup .el-input,
#addRuleDiv .queryGroup input {
	width: 300px !important; 
}
</style>
<div id='addRuleDiv' style='margin-left:20px;margin-top:20px;width: 97%;'>
	<el-form ref='ruleForm' :model='ruleForm' :rules='rules' label-position="left" label-width='100px'>
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("GuiZeMingCheng")%>' style='margin:20px 0px 25px 25px;display:inline-block' prop="tempName">
			<el-input v-model='ruleForm.tempName' :disabled="viewRule" style='width:200px;'></el-input>
		</el-form-item>
		<el-form-item label="<%=rb.getString("ZhuangTai")%>" style='display:inline-block;vertical-align:bottom;margin-left:150px;'  prop="status" label-width="70px">
			<el-radio-group v-model='ruleForm.status' :disabled="viewRule" style='margin-top:8px;'>
				<el-radio label='1' style='margin-right:45px;'><%=rb.getString("QiYong")%></el-radio>
				<el-radio label='0'><%=rb.getString("JinYong")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("TiaoJianSheZhi")%></span>
		</div>
		<div style='margin-left:25px;width:50%;margin-bottom:20px;position:relative'>
			<div style='margin-bottom:14px;margin-top:20px;margin-right:25px;'>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<div @click='addControlType("sn")' v-show="!viewRule" class="circleIcon placeholder-bt" style="top: 0px;right:60px;" placeholder="<%=rb.getString("TianJia")%>">		
					<span class="el-icon el-icon el-icon-circle-add"></span>
				</div>
				<div @click='importControlType("sn")' v-show="!viewRule" class="circleIcon placeholder-bt" style="top: 0px;right:22px;" placeholder="<%=rb.getString("DaoRu")%>">		
					<span class="el-icon el-icon el-icon-circle-import"></span>
				</div>
			</div>
			<div style='height:320px;border:1px solid #DEDFE6;margin-right:25px;'>
				<el-ctable ref="ctableSetting" :url='settingUrl' :height="height" :readonly="viewRule"
					:row-key="'serialNumber'" :query-params="params_setting" pagination="true" :rownumber=true
					@load-success="loadSuccessEnb"
					>
					
					<!-- 模糊查询 -- 接入规则 -->
					<template slot="toolbar">
						<div class='queryGroup'> 
							<el-input v-model='params_setting_form.searchText' @keyup.enter.native="queryEnb" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
							<i @click='queryEnb' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
						</div>
						<p @click="clearType('sn')" v-show="!viewRule" style='float:right;margin:10px 10px 0;'><span class='el-icon el-icon-operation-delete'></span>Clear</p>
					</template>
		
					<!-- 主列表 -->
					<el-table-column type='selection' width="50"></el-table-column>
					<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber"></el-table-column>
				</el-ctable>
			</div>
			<p style='margin-top:5px;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span><%=rb.getString("JieRuKongZhiTianJiaJiZhanTiShi")%></p>
		</div>
		<div class="group-title not-extend">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("KongZhiFangShi")%></span>
		</div>
		<div style='margin-left:25px;width:98%;'>
			<el-form-item label="<%=rb.getString("KongZhiFangShi")%>" style='margin:10px 0px'>
				<el-checkbox-group :disabled="viewRule" v-model='checkList' style='margin-top:5px;' @change="changeControlType">
					<el-checkbox label='tac'>TAC</el-checkbox>
					<el-checkbox label='ecgi'>ECGI</el-checkbox>
					<el-checkbox label='ip'>IP</el-checkbox>
					<el-checkbox label='gps'>GPS Position</el-checkbox>
				</el-checkbox-group>
			</el-form-item>
			<div v-show="showType" style='display:grid;grid-template-columns:repeat(auto-fill,48%);grid-template-rows:repeat(auto-fill,350px);grid-column-gap:20px;width:90%;grid-row-gap:30px;'>
				<!-- TAC -->
				<div class='typeItem' v-if="showTac">
					<el-ctable :readonly="viewRule" :height="height" :url='tacUrl' ref="ctableTac" @load-success="loadSuccessTac"
							:row-key="'tac'" :query-params="params_tac" pagination="true" :rownumber=true>
							
						<!-- 模糊查询  -->
						<template slot="toolbar">
							<div  style='margin-top:5px; position: relative;'>
								<span style='font-size:14px;font-weight:bold;margin-left:20px;vertical-align:text-bottom'>TAC</span>
								<el-form-item style="display:inline-block;margin-bottom:15px;" prop="neighborAccess" label-width="20px">
									<el-checkbox v-model="ruleForm.neighborAccess" :disabled="viewRule" style='vertical-align:text-bottom'><%=rb.getString("WuLinQuJieRu")%></el-checkbox>
								</el-form-item>
								<div>
									<div class="newIconBoxCls-bt" @click='addControlType("tac")' v-show="!viewRule" style="right:82px;top:0px;" tip="<%=rb.getString("TianJia")%>">
										<span class="el-icon el-icon-circle-add" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click='importControlType("tac")' v-show="!viewRule" style="right:46px;top:0px;" tip="<%=rb.getString("DaoRu")%>">
										<span class="el-icon el-icon-circle-import" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click="exportData('tac')" style="right:10px;top:0px;" tip="<%=rb.getString("DaoChu")%>">
										<span class="el-icon el-icon-circle-export" ></span>
									</div>									
								</div>
							</div>
							<div class='queryGroup'> 
								<el-input v-model='params_tac_form.searchText' @keyup.enter.native="queryTac" class='pairgrid-query' placeholder='TAC'></el-input>
								<i @click='queryTac' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
							</div>
							<p @click="clearType('tac')" v-show="!viewRule" style='float:right;margin:10px 10px 0;'><span class='el-icon el-icon-operation-delete'></span>Clear</p>
						</template>
			
						<!-- 主列表 -->
						<el-table-column type='selection' width="50"></el-table-column>
						<el-table-column label='TAC' prop="tac"></el-table-column>
					</el-ctable>
					<p style='margin-top:5px;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span><%=rb.getString("JieRuKongZhiTianJiaTACTiShi")%></p>
				</div>
				<!-- ECGI -->
				<div class='typeItem' v-if="showEcgi">
					<el-ctable :readonly="viewRule"  :height="height" ref="ctableEcgi" :url="ecgiUrl" @load-success="loadSuccessEcgi"
							:row-key="'ecgi'" :query-params="params_ecgi" pagination="true" :rownumber=true>
							
						<!-- 模糊查询  -->
						<template slot="toolbar">
							<div style='margin-top:5px; position: relative;'>
								<span style='font-size:14px;font-weight:bold;margin-left:20px;vertical-align:text-bottom'>ECGI</span>
								<el-form-item style="display:inline-block;margin-bottom:15px;" prop="neighborAccess" label-width="20px">
									<el-checkbox v-model="ruleForm.neighborAccess" :disabled="viewRule" style='vertical-align:text-bottom'><%=rb.getString("WuLinQuJieRu")%></el-checkbox>
								</el-form-item>
								<div>
									<div class="newIconBoxCls-bt" @click='addControlType("ecgi")' v-show="!viewRule" style="right:82px;top:0px;" tip="<%=rb.getString("TianJia")%>">
										<span class="el-icon el-icon-circle-add" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click='importControlType("ecgi")' v-show="!viewRule" style="right:46px;top:0px;" tip="<%=rb.getString("DaoRu")%>">
										<span class="el-icon el-icon-circle-import" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click="exportData('ecgi')" style="right:10px;top:0px;" tip="<%=rb.getString("DaoChu")%>">
										<span class="el-icon el-icon-circle-export" ></span>
									</div>									
								</div>
							</div>
							<div class='queryGroup'> 
								<el-input v-model="params_ecgi_form.searchText" @keyup.enter.native="queryEcgi" class='pairgrid-query' placeholder='ECGI'></el-input>
								<i @click='queryEcgi' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
							</div>
							<p @click="clearType('ecgi')" v-show="!viewRule" style='float:right;margin:10px 10px 0;'><span class='el-icon el-icon-operation-delete'></span>Clear</p>
						</template>
			
						<!-- 主列表 -->
						<el-table-column type='selection' width="50"></el-table-column>
						<el-table-column label='ECGI' prop="ecgi"></el-table-column>
					</el-ctable>
					<p style='margin-top:5px;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span><%=rb.getString("JieRuKongZhiTianJiaECGITiShi")%></p>
				</div>
				<!-- IP -->
				<div class='typeItem' v-if="showIp">
					<el-ctable :readonly="viewRule" :height="height" ref="ctableIp" :url="ipUrl" @load-success="loadSuccessIp"
							:row-key="'id'" :query-params="params_ip" pagination="true" :rownumber=true>
							
						<!-- 模糊查询  -->
						<template slot="toolbar">
							<div style='margin:5px 0px 15px 0px; position: relative;'>
								<span style='font-size:14px;font-weight:bold;margin-left:20px;vertical-align:text-bottom'>IP</span>
								<div>
									<div class="newIconBoxCls-bt" @click='addControlType("ip")' v-show="!viewRule" style="right:82px;top:0px;" tip="<%=rb.getString("TianJia")%>">
										<span class="el-icon el-icon-circle-add" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click='importControlType("ip")' v-show="!viewRule" style="right:46px;top:0px;" tip="<%=rb.getString("DaoRu")%>">
										<span class="el-icon el-icon-circle-import" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click="exportData('ip')" style="right:10px;top:0px;" tip="<%=rb.getString("DaoChu")%>">
										<span class="el-icon el-icon-circle-export" ></span>
									</div>									
								</div>
							</div>
							<div class='queryGroup'> 
								<el-input v-model="params_ip_form.searchText" @keyup.enter.native="queryIp" class='pairgrid-query' placeholder='<%=rb.getString("IPDiZhi")%>'></el-input>
								<i @click="queryIp" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
							</div>
							<p @click="clearType('ip')" v-show="!viewRule" style='float:right;margin:10px 10px 0;'><span class='el-icon el-icon-operation-delete'></span>Clear</p>
						</template>
			
						<!-- 主列表 -->
						<el-table-column type='selection' width="50"></el-table-column>
						<el-table-column label='Start IP' prop="startIp"></el-table-column>
						<el-table-column label='End IP' prop="endIp"></el-table-column>
					</el-ctable>
					<p style='margin-top:5px;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span><%=rb.getString("JieRuKongZhiTianJiaIPTiShi")%></p>
				</div>
				<!-- GPS -->
				<div class='typeItem' v-if="showGps">
					<el-ctable :readonly="viewRule" :height="height" ref="ctableGps" :url="gpsUrl" @load-success="loadSuccessGps"
							:row-key="'id'" :query-params="params_gps" pagination="true" :rownumber=true>
							
						<!-- 模糊查询 -- 接入规则 -->
						<template slot="toolbar">
							<div style='margin-top:5px; position: relative;'>
								<span style='font-size:14px;font-weight:bold;margin-left:20px;vertical-align:text-bottom'>GPS Position</span>
								<el-form-item style="display:inline-block;margin-bottom:15px;" prop="gpsAccessEnable" label-width="20px">
									<el-checkbox v-model="ruleForm.gpsAccessEnable" :disabled="viewRule" style='vertical-align:text-bottom'><%=rb.getString("WuGPSJieRu")%></el-checkbox>
								</el-form-item>
								<div>
									<div class="newIconBoxCls-bt" @click='addControlType("gps")' v-show="!viewRule" style="right:82px;top:0px;" tip="<%=rb.getString("TianJia")%>">
										<span class="el-icon el-icon-circle-add" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click='importControlType("gps")' v-show="!viewRule" style="right:46px;top:0px;" tip="<%=rb.getString("DaoRu")%>">
										<span class="el-icon el-icon-circle-import" ></span>
									</div>
									<div class="newIconBoxCls-bt" @click="exportData('gps')" style="right:10px;top:0px;" tip="<%=rb.getString("DaoChu")%>">
										<span class="el-icon el-icon-circle-export" ></span>
									</div>									
								</div>
							</div>
							<div class='queryGroup'> 
								<el-input v-model="params_gps_form.searchText" class='pairgrid-query' @keyup.enter.native="queryGps" placeholder='<%=rb.getString("GPSJingDu")%>/<%=rb.getString("GPSWeiDu")%>'></el-input>
								<i @click="queryGps" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
							</div>
							<p @click="clearType('gps')" v-show="!viewRule" style='float:right;margin:10px 10px 0;'><span class='el-icon el-icon-operation-delete'></span>Clear</p>
						</template>
			
						<!-- 主列表 -->
						<el-table-column type='selection' width="50"></el-table-column>
						<el-table-column label='<%=rb.getString("JingDuFanWei")%>' prop="longitudeRange"></el-table-column>
						<el-table-column label='<%=rb.getString("WeiDuFanWei")%>' prop="latitudeRange"></el-table-column>
					</el-ctable>
					<p style='margin-top:5px;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;margin-right:5px;'></span><%=rb.getString("JieRuKongZhiTianJiaGPSTiShi")%></p>
				</div>
			</div>
		</div>
		<el-form-item prop='typeTest' style='margin-top:20px;' label-width='20px'>
			<el-input v-model='ruleForm.typeTest' v-show=false></el-input>
		</el-form-item>
		<el-form-item prop='enbStr' style='margin-bottom:0px;'>
			<el-input v-model='ruleForm.enbStr' v-show=false></el-input>
		</el-form-item>
		<el-form-item prop='tacStr' style='margin-bottom:0px;'>
			<el-input v-model='ruleForm.tacStr' v-show=false></el-input>
		</el-form-item>
		<el-form-item prop='ecgiStr' style='margin-bottom:0px;'>
			<el-input v-model='ruleForm.ecgiStr' v-show=false></el-input>
		</el-form-item>
		<el-form-item prop='ipStr' style='margin-bottom:0px;'>
			<el-input v-model='ruleForm.ipStr' v-show=false></el-input>
		</el-form-item>
		<el-form-item prop='gpsStr' style='margin-bottom:0px;'>
			<el-input v-model='ruleForm.gpsStr' v-show=false></el-input>
		</el-form-item>
	</el-form>
	<el-dialog class='dialogStyle accessControlDialogCls' :title='addTitle' width='600px' :visible.sync='addTypeVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeAddType'>
		<el-form ref='addTypeForm' :rules='addTypeRules' :model='addTypeForm' label-position="left" style='margin-top:10px;'>
			<div v-if='showAddType == "sn"'>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;" label='<%=rb.getString("XiaoZhanBianMa")%>' label-width="105px">
					<el-input v-model='addTypeForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB;margin-left:100px;margin-bottom:45px;'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<!-- TAC -->
			<div v-if='showAddType == "tac"'>
				<el-form-item label="TAC" label-width="50px" style='margin-bottom:0px;position:relative'>
					<el-input v-model='addTypeForm.tacValue' style='width:300px;'></el-input>
					<span @click='addTac' class='form-bt el-icon el-icon-plus' style='position:absolute;left:265px;top:-5px;'></span>
				</el-form-item>
				<div style='margin-left:50px;height:150px;overflow:auto'>
					<el-form-item prop='tacGroup' class='suffixItem' v-for='(domain,index) in addTypeForm.tacGroup' style='line-height:16px;'>
						<div class='form-suffix'>
							<span class='text'>{{domain}}</span>
							<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeTac(domain)'></span>
						</div>
					</el-form-item>
					<p style='color:red;font-size:12px;'>{{errorMessage}}</p>
					<el-form-item prop='itemTest' style='margin-bottom:5px;'>
						<el-input v-model='addTypeForm.itemTest'  v-show=false></el-input>
					</el-form-item>
				</div>
			</div>
			
			<!-- ECGI -->
			<div v-if='showAddType == "ecgi"'>
				<el-form-item label="ECGI" label-width="50px" style='margin-bottom:0px;position:relative'>
					<el-input v-model='addTypeForm.ecgiValue' style='width:300px;'></el-input>
					<span @click='addEcgi' class='form-bt el-icon el-icon-plus' style='position:absolute;left:265px;top:-5px;'></span>
				</el-form-item>
				<div style='margin-left:50px;height:150px;overflow:auto'>
					<el-form-item class='suffixItem' v-for='(domain,index) in addTypeForm.ecgiGroup' style='line-height:16px;'>
						<div class='form-suffix'>
							<span class='text'>{{domain}}</span>
							<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeEcgi(domain)'></span>
						</div>
					</el-form-item>
					<p style='color:red;font-size:12px;'>{{errorMessage}}</p>
					<el-form-item prop='itemTest' style='margin-bottom:5px;'>
						<el-input v-model='addTypeForm.itemTest' v-show=false></el-input>
					</el-form-item>
				</div>
			</div>
			
			<!-- IP -->
			<div v-if='showAddType == "ip"' class="gpsItem">
				<el-form-item label="IP" label-width="50px" style='margin-bottom:0px;position:relative'>
					<el-input v-model='addTypeForm.ipStart' style='width:200px;'></el-input>&nbsp;&nbsp;—&nbsp;&nbsp;<el-input v-model='addTypeForm.ipEnd' style='width:200px;'></el-input>
					<span @click='addIp' class='form-bt el-icon el-icon-plus' style='vertical-align:middle'></span>
				</el-form-item>
				<div style='margin-left:50px;height:150px;overflow:auto'>
					<el-form-item class='suffixItem' v-for='(domain,index) in addTypeForm.ipGroup' style='line-height:16px;'>
						<div class='form-suffix'>
							<span class='text'>{{domain}}</span>
							<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeIp(domain)'></span>
						</div>
					</el-form-item>
					<p style='color:red;font-size:12px;'>{{errorMessage}}</p>
					<el-form-item prop='itemTest' style='margin-bottom:5px;'>
						<el-input v-model='addTypeForm.itemTest' v-show=false></el-input>
					</el-form-item>
				</div>
			</div>
			
			<!-- GPS -->
			<div v-if='showAddType == "gps"' class="gpsItem">
				
					<el-form-item label="Longitude Range" label-width='130px' style='margin-bottom:20px;'>
						<el-input v-model='addTypeForm.longitudeStart' style='width:150px;'></el-input>&nbsp;&nbsp;—&nbsp;&nbsp;<el-input v-model='addTypeForm.longitudeEnd' style='width:150px;'></el-input>
					</el-form-item>
					<el-form-item label="Latitude Range" label-width='130px'>
						<el-input v-model='addTypeForm.latitudeStart' style='width:150px;'></el-input>&nbsp;&nbsp;—&nbsp;&nbsp;<el-input v-model='addTypeForm.latitudeEnd' style='width:150px;'></el-input>
					</el-form-item>
					<span @click='addGps' class='form-bt el-icon el-icon-plus' style='position:absolute;right:60px;top:90px;'></span>
					<div style='margin-left:120px;height:150px;overflow:auto'>
						<el-form-item class='suffixItem' v-for='(domain,index) in addTypeForm.gpsGroup' style='line-height:16px;'>
							<div class='form-suffix'>
								<span class='text'>{{domain}}</span>
								<span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='removeGps(domain)'></span>
							</div>
						</el-form-item>
						<p style='color:red;font-size:12px;'>{{errorMessage}}</p>
						<el-form-item prop='itemTest' style='margin-bottom:5px;'>
							<el-input v-model='addTypeForm.itemTest' v-if=false></el-input>
						</el-form-item>
					</div>
				
			</div>
			<div>
				<el-button @click='saveAddType' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeAddType'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
	<el-dialog class='dialogStyle' :title='importTitle' width='650px' :visible.sync='importTypeVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeImportType'>
		<el-form ref='importTypeForm' :rules='importTypeRules' :model='importTypeForm' label-position="left">
			<el-form-item label='<%=rb.getString("DaoRuLeiXing")%>' style='margin-top:20px;' label-width="90px">
				<el-radio-group v-model="importTypeForm.import_type" style='margin-top:8px;'>
					<el-radio label='append'>Append</el-radio>
					<el-radio label='cover'>Cover</el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item label='<%=rb.getString("DaoRuWenJian")%>' style='margin-top:20px;' prop="file_path" label-width="90px">
				<el-input :disabled="true" style='width:300px;' v-model='importTypeForm.file_path'>
					<i @click='importFile' slot='suffix' style='display:inline-block;width:28px;height:28px;margin:4px -9px 0 0;' class='el-icon el-icon-operation-import'></i>
				</el-input>
				<span style='color:#BBB;margin-left:5px;'>Only .xls , .xlsx and .csv are supported.</span>
				<p>
					<span class='el-icon el-icon-circle-info' style='font-size:14px;'></span>
					<span style="color:#BBB"><%=rb.getString("DaoRuWenJianTiShi")%></span>
					<span class='el-icon el-icon-common-download'></span>
					<span @click="exportTemp" style='text-decoration:underline;cursor:pointer'><%=rb.getString("DaoChuMuBan")%></span>
				</p>
			</el-form-item>
			<div style='margin-top:45px;'>
				<el-button @click='saveImportType' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeImportType'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
	<el-dialog class='resultDialog'  title='<%=rb.getString("DaoRuJieGuo")%>' width='630px' :visible.sync='resultVisible' :append-to-body="true" :close-on-click-modal="false">
		<p><%=rb.getString("ZongLiang")%>：<span>{{count}}</span></p>
		<p><%=rb.getString("DaoRuChengGongShu")%>：<span style='color:#38C846'>{{legalCount}}</span></p>
		<p><%=rb.getString("DaoRuShiBaiShu")%>：<span style='color:#E88282'>{{unlegalCount}}</span></p>
		<div style='width:590px;height:300px;border:1px solid #E4E7EC;margin-top:10px;'>
			<el-ctable ref="ctableResult" :url="resultUrl" :query-params = "params_result" height="100%" pagination="true" :rownumber=true>
				<el-table-column :label='resultLabel' :prop="resultProp" width="200"></el-table-column>
				<el-table-column :label='resultLabelEnd' :prop="resultPropEnd" width="200" v-if="resultEnd"></el-table-column>
				<el-table-column label='Status' prop="status" :formatter="statusFmt"></el-table-column>
			</el-ctable>
		</div>
		<div style='margin-top:45px;'>
			<el-button @click="saveResult" type="primary"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="cancelResult"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
	
</div>
<%-- 表单-上传基站列表文件 --%>
<form enctype="multipart/form-data" method="post" id="uploadForm_access" style="display: none;">
	<input name="fileSize"  value="" hidden="true">
    <input name="uploadFile" type="file" id="uploadFileAccess">
    <input name="operType" value="">
</form>
<script>
	var addAccessVue = new Vue({
		el:'#addRuleDiv',
		data(){
			var vm = this;
			//校验基站编码
			var validatorNum = (rule,value,callback) => {
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
			//校验至少添加一个控制类型
			var validateItem = (rule,value,callback) => {
				if(vm.showAddType == 'tac'){
					if(vm.addTypeForm.tacGroup.length == 0){
						callback(new Error("<%=rb.getString("ZhiShaoTianJiaYiGe")%>"))	
					}else{
						callback();
					}
				}
				if(vm.showAddType == 'ecgi'){
					if(vm.addTypeForm.ecgiGroup.length == 0){
						callback(new Error("<%=rb.getString("ZhiShaoTianJiaYiGe")%>"))	
					}else{
						callback();
					}
				}
				if(vm.showAddType == 'ip'){
					if(vm.addTypeForm.ipGroup.length == 0){
						callback(new Error("<%=rb.getString("ZhiShaoTianJiaYiGe")%>"))	
					}else{
						callback();
					}
				}
				if(vm.showAddType == 'gps'){
					if(vm.addTypeForm.gpsGroup.length == 0){
						callback(new Error("<%=rb.getString("ZhiShaoTianJiaYiGe")%>"))	
					}else{
						callback();
					}
				}
				
			};
			//校验文件路径
			var validateFilePath = (rule,value,callback) => {
				if(value == ""){
					callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
				}else if(!fileFormatMatch(value,"xlsx,xls,csv")){
					callback(new Error("<%=rb.getString("DaoRuWenJianGeShi")%>"))
				}else{
					callback();
				}
			};
			//校验控制类型列表是否为空
			var validateType = (rule,value,callback) => {
				if(vm.checkList.length == 0){
					callback(new Error('<%=rb.getString("QingXuanZeTiaoJianGuiZe")%>'))
				}
				if(vm.checkList.includes('tac') && vm.ruleForm.tacStr == ''){
					callback(new Error('<%=rb.getString("QingTianJiaTAC")%>'))
				}
				if(vm.checkList.includes('ecgi') && vm.ruleForm.ecgiStr == ''){
					callback(new Error('<%=rb.getString("QingTianJiaECGI")%>'))
				}
				if(vm.checkList.includes('ip') && vm.ruleForm.ipStr == ''){
					callback(new Error('<%=rb.getString("QingTianJiaIP")%>'))
				}
				if(vm.checkList.includes('gps') && vm.ruleForm.gpsStr == ''){
					callback(new Error('<%=rb.getString("QingTianJiaGPS")%>'))
				}
				callback();
			};
			return{
				ruleForm:{
					tempName:'',
					status:'1',
					tacEnable:false,
					ecgiEnable:false,
					ipEnable:false,
					gpsEnable:false,
					neighborAccess:false,
					gpsAccessEnable:false,
					typeTest:'',
					enbStr:'',
					tacStr:'',
					ipStr:'',
					ecgiStr:'',
					gpsStr:''
				},
				rules:{
					tempName:[
						{required:true,message:'<%=rb.getString("QingShuRuGuiZeMingCheng")%>',trigger:'blur'},
						{min:0,max:64,message:'<%=rb.getString("ZiFuChang")%>:1-64',trigger:'blur'}
					],
					typeTest:[
						{validator:validateType}
					]
				},
				height:'100%',
				settingUrl:'${ctx}/son/access/queryAccessTempSNPageList.action',
				params_setting:{
					tempId:"",
					searchText:'',
					status:'active'
				},
				params_setting_form:{searchText:''},
				tacUrl:'${ctx}/son/access/queryAccessTempTACPageList.action',
				ecgiUrl:'${ctx}/son/access/queryAccessTempECGIPageList.action',
				ipUrl:"${ctx}/son/access/queryAccessTempIpPageList.action",
				gpsUrl:"${ctx}/son/access/queryAccessTempGPSPageList.action",
				params_tac:{
					tempId:'',
					searchText:''
				},
				params_tac_form:{searchText:""},
				params_ecgi:{
					tempId:'',
					searchText:''
				},
				params_ecgi_form:{searchText:""},
				params_ip:{
					tempId:'',
					searchText:''
				},
				params_ip_form:{searchText:""},
				params_gps:{
					tempId:'',
					searchText:''
				},
				params_gps_form:{searchText:""},
				params_result:{
					tempId:'',
					searchText:'',
					status:'import'
				},
				addTypeVisible:false,
				addTypeForm:{
					serialNumber:'',
					tacValue:'',
					tacGroup:[],
					ecgiValue:'',
					ecgiGroup:[],
					ipStart:'',
					ipEnd:'',
					ipGroup:[],
					longitudeStart:'',
					longitudeEnd:'',
					latitudeStart:'',
					latitudeEnd:'',
					gpsGroup:[],
					itemTest:''
				},
				addTypeRules:{
					serialNumber:[
						{validator:validatorNum,trigger:'blur'}
					],
					itemTest:[
						{validator:validateItem}
					],
					file_path:[
						{validator:validateFilePath}
					]
				},
				handTypeFlag:true,
				errorMessage:'',
				showAddType:'',
				addTitle:'',
				resultVisible:false,
				resultUrl:'',
				count:'',
				legalCount:'',
				unlegalCount:'',
				resultLabel:'',
				resultLabelEnd:'',
				result_params:{
					operType : '',
					importType : '',
					rd: ''
				},
				resultTarget:'',
				resultProp:'',
				resultPropEnd:'',
				resultEnd:false,
				tempId:'',
				viewRule:false,
				enbFlag:true,
				tacFlag:true,
				ecgiFlag:true,
				ipFlag:true,
				gpsFlag:true,
				importTitle:'',
				importTypeVisible:false,
				importTypeForm:{
					import_type:'append',
					file_path:'',
				},
				importTypeRules:{
					file_path:[
						{validator:validateFilePath}
					]
				},
				checkList:[],
				oldControlType:[]
			}
		},
		methods:{
			//基站列表模糊查询
			queryEnb(){
				Object.assign(this.params_setting,this.params_setting_form)
			},
			//tac列表模糊查询
			queryTac(){
				Object.assign(this.params_tac,this.params_tac_form)
			},
			//ecgi列表模糊查询
			queryEcgi(){
				Object.assign(this.params_ecgi,this.params_ecgi_form)
			},
			//ip列表模糊查询
			queryIp(){
				Object.assign(this.params_ip,this.params_ip_form)
			},
			//gps列表模糊查询
			queryGps(){
				Object.assign(this.params_gps,this.params_gps_form)
			},
			//基站列表加载完成方法
			loadSuccessEnb(){
				var vm = this;
				vm.$refs.ctableSetting.clearSelection();
				if(vm.enbFlag){
					vm.ruleForm.enbStr = vm.$refs.ctableSetting.getData().map(function(item){
						return item.serialNumber;
					}).toString();
					initForm(vm.$refs.ruleForm);
					vm.enbFlag = false;
				}else{
					vm.ruleForm.enbStr = vm.$refs.ctableSetting.getData().map(function(item){
						return item.serialNumber;
					}).toString();
				}
			},
			//tac列表加载完成方法
			loadSuccessTac(){
				var vm = this;
				vm.$refs.ctableTac.clearSelection();
				if(vm.tacFlag){
					vm.ruleForm.tacStr = vm.$refs.ctableTac.getData().map(function(item){
						return item.tac;
					}).toString();
					initForm(vm.$refs.ruleForm);
					vm.tacFlag = false;
				}else{
					vm.ruleForm.tacStr = vm.$refs.ctableTac.getData().map(function(item){
						return item.tac;
					}).toString();
				}
			},
			//ecgi列表加载完成方法
			loadSuccessEcgi(){
				var vm = this;
				vm.$refs.ctableEcgi.clearSelection();
				if(vm.ecgiFlag){
					vm.ruleForm.ecgiStr = vm.$refs.ctableEcgi.getData().map(function(item){
						return item.ecgi;
					}).toString();
					initForm(vm.$refs.ruleForm);
					vm.ecgiFlag = false;
				}else{
					vm.ruleForm.ecgiStr = vm.$refs.ctableEcgi.getData().map(function(item){
						return item.ecgi;
					}).toString();
				}
			},
			//ip列表加载完成方法
			loadSuccessIp(){
				var vm =  this;
				vm.$refs.ctableIp.clearSelection();
				if(vm.ipFlag){
					vm.ruleForm.ipStr = vm.$refs.ctableIp.getData().map(function(item){
						return item.startIp+"-"+item.endIp;
					}).toString();
					initForm(vm.$refs.ruleForm);
					vm.ipFlag = false
				}else{
					vm.ruleForm.ipStr = vm.$refs.ctableIp.getData().map(function(item){
						return item.startIp+"-"+item.endIp;
					}).toString();
				}
			},
			//gps列表加载完成方法
			loadSuccessGps(){
				var vm = this;
				vm.$refs.ctableGps.clearSelection();
				if(vm.gpsFlag){
					vm.ruleForm.gpsStr = vm.$refs.ctableGps.getData().map(function(item){
						return item.longitudeRange+";"+item.latitudeRange;
					}).toString();
					initForm(vm.$refs.ruleForm);
					vm.gpsFlag = false;
				}else{
					vm.ruleForm.gpsStr = vm.$refs.ctableGps.getData().map(function(item){
						return item.longitudeRange+";"+item.latitudeRange;
					}).toString();
				}
			},
			//导入文件
			importFile(){
				$("#uploadForm_access input[name='uploadFile']").click();
			},
			/**
			 * 打开添加弹窗并控制显示的模块
			 * @param type {string} 类型
			*/
			addControlType(type){
				var typeObj = {
					'sn' : 'Add eNB',
					'tac' : 'Add TAC',
					'ecgi' : 'Add ECGI',
					'ip' : 'Add IP',
					'gps' : 'Add GPS'
				}
				this.showAddType = type;
				this.addTitle = typeObj[type];
				this.addTypeVisible = true;
			},
			//添加tac
			addTac(){
				var vm = this,
					msg = "<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 0<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 65535",
					reg = /^[0-9]\d*$/,
					value = vm.addTypeForm.tacValue;
				if(value != '' && reg.test(value) && value >= 0 && value <=65535){
					if(vm.addTypeForm.tacGroup.indexOf(value) == -1){
						vm.addTypeForm.tacGroup.push(value);
						vm.addTypeForm.tacValue = '';
						vm.errorMessage = '';
						vm.$refs.addTypeForm.validateField('itemTest');
					}else{
						vm.errorMessage = '<%=rb.getString("YiCunZai")%>';
					}
				}else{
					vm.errorMessage = msg;
				}
			},
			/**
			 * 删除tac
			 * @param item {string} 删除项
			*/
			removeTac(item){
				var vm = this;
				var index = vm.addTypeForm.tacGroup.indexOf(item);
				if(index !== -1){
					vm.addTypeForm.tacGroup.splice(index,1)
				}
				vm.errorMessage = '';
			},
			//添加ecgi
			addEcgi(){
				var vm = this,
					msg = "<%=rb.getString("ECGITiShi")%>",
					reg = /^[1-9]{1}\d{4,5}:\d+$/,
					value = vm.addTypeForm.ecgiValue,
					index = value.indexOf(':'),
					eci = value.substring(index+1);
				if(value != '' && reg.test(value) && eci >=0 && eci <= 268435455){
					if(vm.addTypeForm.ecgiGroup.indexOf(value) == -1){
						vm.addTypeForm.ecgiGroup.push(value);
						vm.addTypeForm.ecgiValue = '';
						vm.errorMessage = '';
						vm.$refs.addTypeForm.validateField('itemTest');
					}else{
						vm.errorMessage = '<%=rb.getString("YiCunZai")%>';
					}
				}else{
					vm.errorMessage = msg;
				}
			},
			/**
			 * 删除ecgi
			 * param item {string} 删除项
			*/
			removeEcgi(item){
				var vm = this;
				var index = vm.addTypeForm.ecgiGroup.indexOf(item);
				if(index !== -1){
					vm.addTypeForm.ecgiGroup.splice(index,1)
				}
				vm.errorMessage = '';
			},
			//添加ip
			addIp(){
				var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
					vm = this,
					ipStart = vm.addTypeForm.ipStart,
					ipEnd = vm.addTypeForm.ipEnd,
					str = '';
				if(ipStart == '' || ipEnd == ''){
					vm.errorMessage = '<%=rb.getString("IPDiZhiBuNengWeiKong")%>';
				}else{
					if(reg.test(ipStart) && reg.test(ipEnd) && vm.compareIp(ipStart,ipEnd)){
						str = ipStart + "-" + ipEnd;
						if(vm.addTypeForm.ipGroup.indexOf(str) == -1){
							vm.addTypeForm.ipGroup.push(str);
							vm.addTypeForm.ipStart = '';
							vm.addTypeForm.ipEnd = '';
							vm.errorMessage = '';
							vm.$refs.addTypeForm.validateField('itemTest');
						}else{
							vm.errorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
						}
					}else{
						vm.errorMessage = '<%=rb.getString("IPDiZhiFeiFa")%>';
					}
				}
			},
			/**
			 * 删除ip
			 * @param item {string} 删除项
			*/
			removeIp(item){
				var vm = this;
				var index = vm.addTypeForm.ipGroup.indexOf(item);
				if(index !== -1){
					vm.addTypeForm.ipGroup.splice(index,1)
				}
				vm.errorMessage = '';
			},
			/**
			 * 比较ip大小
			 * @param ipStart {string} 初始ip
			 * @param ipEnd {string} 结束ip
			*/
			compareIp(ipStart,ipEnd){
				var temp1,
					temp2,
					bool = true;
				temp1 = ipStart.split(".");
				temp2 = ipEnd.split(".");
				for(var i=0;i<4;i++){
					if(parseInt(temp1[i])>parseInt(temp2[i])){
						bool = false;
					}
				}

				return bool;
			},
			//添加gps
			addGps(){
				var vm = this,
					longitudeStart = vm.addTypeForm.longitudeStart,
					longitudeEnd = vm.addTypeForm.longitudeEnd,
					latitudeStart = vm.addTypeForm.latitudeStart,
					latitudeEnd = vm.addTypeForm.latitudeEnd,
					//reg = /^[+-]?\d+(\.(\d){0,6})?$|^$|^(\d+|\-){7,}$/,
					reg = /^[+-]?(\d+|\d+(\.)\d{0,6})$/,
					longitudeMinVal = -180,
					longitudeMaxVal = 180,
					latitudeMinVal = -90,
					latitudeMaxVal = 90,
					longitudeArr = [],
					latitudeArr = [];
					str = longitudeStart + "~" + longitudeEnd + ";" + latitudeStart + "~" + latitudeEnd;
				if(longitudeStart == "" || longitudeEnd == "" || latitudeStart == "" || latitudeEnd == ""){
					vm.errorMessage = "<%=rb.getString("FanWeiBuNengWeiKong")%>";
					return;
				}
				if(!(reg.test(longitudeStart) && parseFloat(longitudeStart) >= longitudeMinVal && parseFloat(longitudeStart) <= longitudeMaxVal && reg.test(longitudeEnd) && parseFloat(longitudeEnd) >= longitudeMinVal && parseFloat(longitudeEnd) <= longitudeMaxVal)){
					vm.errorMessage = "longitude <%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>" + longitudeMinVal.toFixed(6) + "-" + longitudeMaxVal.toFixed(6);
					return;
				}
				if(!(reg.test(latitudeStart) && parseFloat(latitudeStart) >= latitudeMinVal && parseFloat(latitudeStart) <= latitudeMaxVal && reg.test(latitudeEnd) && parseFloat(latitudeEnd) >= latitudeMinVal && parseFloat(latitudeEnd) <= latitudeMaxVal)){
					vm.errorMessage = "latitude <%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%>" + latitudeMinVal.toFixed(6) + "-" + latitudeMaxVal.toFixed(6);
					return;
				}
				if(parseFloat(longitudeStart) > parseFloat(longitudeEnd) || parseFloat(latitudeStart) > parseFloat(latitudeEnd)){
					vm.errorMessage = "<%=rb.getString("FanWeiZuiDaZhiDaYuZuiXiaoZhi")%>";
					return;
				}
				if(vm.addTypeForm.gpsGroup.length == 0){
					vm.addTypeForm.gpsGroup.push(str);
					vm.errorMessage = "";
					vm.addTypeForm.longitudeStart = "";
					vm.addTypeForm.longitudeEnd = "";
					vm.addTypeForm.latitudeStart = "";
					vm.addTypeForm.latitudeEnd = "";
					vm.$refs.addTypeForm.validateField('itemTest');
				}else{
					vm.addTypeForm.gpsGroup.map(function(item){
						longitudeArr.push(item.split(";")[0]);
						latitudeArr.push(item.split(";")[1]);
					})
					var longitudeStr = longitudeStart + "~" + longitudeEnd;
					var latitudeStr = latitudeStart + "~" + latitudeEnd;
					if(vm.addTypeForm.gpsGroup.indexOf(str) != -1){
						vm.errorMessage = "latitude <%=rb.getString("YiCunZai")%>";
						return;
					}
					var mutexFlaglongitude = true;
					var mutexFlaglatitude = true;
					longitudeArr.map(function(item,index){
						if(parseFloat(longitudeEnd) <item.split("~")[0] || parseFloat(longitudeStart) > item.split("~")[1]){
							
						}else{
							mutexFlaglongitude = false;
						}
					})
					latitudeArr.map(function(item,index){
						if(parseFloat(latitudeEnd) <item.split("~")[0] || parseFloat(latitudeStart) > item.split("~")[1]){
							
						}else{
							mutexFlaglatitude = false;
						}
					})
					if(!mutexFlaglongitude){
						vm.errorMessage = '<%=rb.getString("JingDuFanWeiHuChi")%>';
						return;
					}
					if(!mutexFlaglatitude){
						vm.errorMessage = '<%=rb.getString("WeiDuFanWeiHuChi")%>';
						return;
					}
					vm.addTypeForm.gpsGroup.push(str);
					vm.errorMessage = "";
					vm.addTypeForm.longitudeStart = "";
					vm.addTypeForm.longitudeEnd = "";
					vm.addTypeForm.latitudeStart = "";
					vm.addTypeForm.latitudeEnd = "";
					vm.$refs.addTypeForm.validateField('itemTest');
				}
			},
			/**
			 * 删除gps
			 * @param item {string} 删除项
			*/
			removeGps(item){
				var vm = this;
				var index = vm.addTypeForm.gpsGroup.indexOf(item);
				if(index !== -1){
					vm.addTypeForm.gpsGroup.splice(index,1)
				}
				vm.errorMessage = '';
			},
			//添加基站或者控制类型的提交方法
			saveAddType(){
				var vm = this;
				vm.$refs.addTypeForm.validate((valid) => {
					if(valid){
						if(vm.showAddType == 'sn'){							
							var serialNumber = vm.addTypeForm.serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
							axios.post("${ctx}/son/access/addAccessTempSN.action",stringify({serialNumber:serialNumber.join(';')})).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$refs.ctableSetting.refresh();
									vm.closeAddType();
								}else{
									vm.$message.error(data["message"])
								}
							}).catch(function(){
								vm.$message.error(data["message"])
							})
						}
						if(vm.showAddType == 'tac'){
							var tac = vm.addTypeForm.tacGroup.toString().split(",").join(";");
							var params = {
								tempId : vm.tempId,
								tac : tac
							}
							axios.post("${ctx}/son/access/addAccessTempTAC.action",stringify(params)).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$refs.ctableTac.refresh();
									vm.closeAddType();
								}else{
									vm.$message.error(data["message"])
								}
							})
						}
						if(vm.showAddType == 'ecgi'){
							var ecgi = vm.addTypeForm.ecgiGroup.toString().split(",").join(";");
							var params = {
								tempId : vm.tempId,
								ecgi : ecgi
							}
							axios.post("${ctx}/son/access/addAccessTempECGI.action",stringify(params)).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$refs.ctableEcgi.refresh();
									vm.closeAddType();
								}else{
									vm.$message.error(data["message"])
								}
							})
						}
						if(vm.showAddType == 'ip'){
							var ip = vm.addTypeForm.ipGroup.toString().split(",").join(";");
							var params = {
								tempId : vm.tempId,
								ipRange : ip
							}
							axios.post("${ctx}/son/access/addAccessTempIp.action",stringify(params)).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$refs.ctableIp.refresh();
									vm.closeAddType();
								}else{
									vm.$message.error(data["message"])
								}
							})
						}
						if(vm.showAddType == 'gps'){
							var longitudeArr = [];
							var latitudeArr = [];
							vm.addTypeForm.gpsGroup.map(function(item){
								longitudeArr.push(item.split(";")[0]);
								latitudeArr.push(item.split(";")[1]);
							})
							var longitudeRange = longitudeArr.toString().split(",").join(";")
							var latitudeRange = latitudeArr.toString().split(",").join(";");
							var params = {
								tempId : vm.tempId,
								longitudeRange : longitudeRange,
								latitudeRange : latitudeRange
							}
							axios.post("${ctx}/son/access/addAccessTempGPS.action",stringify(params)).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$refs.ctableGps.refresh();
									vm.closeAddType();
								}else{
									vm.$message.error(data["message"])
								}
							})
						}
					}
				})
			},
			saveImportType(){
				var vm = this;
				vm.$refs.importTypeForm.validate((valid) => {
					if(valid){
						var files = document.querySelector("#uploadFileAccess").files;
						$("#uploadForm_access [name=fileSize]").val(files[0].size);
			    		var resultUrl = ""
						if(vm.showAddType == 'sn'){
							vm.resultLabel = "Serial Number";
							vm.resultProp = "serialNumber";
							vm.resultEnd = false;
							Object.assign(vm.result_params,{
								operType : 'enb',
								importType : vm.importTypeForm.import_type,
								rd: Math.random()
							})
							resultUrl = "${ctx}/son/access/queryAccessTempSNPageList.action"
				    		$("#uploadForm_access [name=operType]").val('enb');
				    		vm.resultTarget = vm.$refs.ctableSetting;
						}
						if(vm.showAddType == 'tac'){
							vm.resultLabel = "TAC";
							vm.resultProp = "tac";
							vm.resultEnd = false;
							Object.assign(vm.result_params,{
								operType : 'tac',
								importType : vm.importTypeForm.import_type,
								rd: Math.random()
							})
							resultUrl = "${ctx}/son/access/queryAccessTempTACPageList.action"
							$("#uploadForm_access [name=operType]").val('tac');
							vm.resultTarget = vm.$refs.ctableTac;
						}
						if(vm.showAddType == 'ecgi'){
							vm.resultLabel = "ECGI";
							vm.resultProp = "ecgi";
							vm.resultEnd = false;
							Object.assign(vm.result_params,{
								operType : 'ecgi',
								importType : vm.importTypeForm.import_type,
								rd: Math.random()
							})
							resultUrl = "${ctx}/son/access/queryAccessTempECGIPageList.action"
							$("#uploadForm_access [name=operType]").val('ecgi');
							vm.resultTarget = vm.$refs.ctableEcgi;
						}
						if(vm.showAddType == 'ip'){
							vm.resultLabel = "Start IP";
							vm.resultLabelEnd = "End IP";
							vm.resultProp = "startIp";
							vm.resultEnd = true;
							vm.resultPropEnd = "endIp";
							Object.assign(vm.result_params,{
								operType : 'ip',
								importType : vm.importTypeForm.import_type,
								rd: Math.random()
							})
							resultUrl = "${ctx}/son/access/queryAccessTempIpPageList.action"
							$("#uploadForm_access [name=operType]").val('ip');
							vm.resultTarget = vm.$refs.ctableIp;
						}
						if(vm.showAddType == 'gps'){
							vm.resultLabel = "Longitude Range";
							vm.resultLabelEnd = "Latitude Range";
							vm.resultProp = "longitudeRange";
							vm.resultEnd = true;
							vm.resultPropEnd = "latitudeRange";
							Object.assign(vm.result_params,{
								operType : 'gps',
								importType : vm.importTypeForm.import_type,
								rd: Math.random()
							})
							resultUrl = "${ctx}/son/access/queryAccessTempGPSPageList.action"
							$("#uploadForm_access [name=operType]").val('gps');
							vm.resultTarget = vm.$refs.ctableGps;
						}
						
						uploadWithProgress({
							url: "${ctx}/son/access/importAccessFile.action",
							form: document.querySelector("#uploadForm_access"),
							success: function (data) {
						     	if(data["success"]){
						     		vm.closeImportType();
						     		$("#uploadForm_access input[name='uploadFile']").val("");
						     		vm.resultVisible = true;
						     		vm.resultUrl = resultUrl;
						     		vm.count = data.count;
						     		vm.legalCount = data.legalCount;
						     		vm.unlegalCount = data.unlegalCount;
									if(vm.$refs.ctableResult){
										vm.$refs.ctableResult.refresh();
									}
						     	}else{
					        		vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
						     	}
						    }
						});
					}
				})
			},
			//关闭添加基站或者控制类型的弹框
			closeAddType(){
				this.addTypeVisible = false;
				this.addTypeForm = {
					serialNumber:'',
					tacValue:'',
					tacGroup:[],
					ecgiValue:'',
					ecgiGroup:[],
					ipStart:'',
					ipEnd:'',
					ipGroup:[],
					longitudeStart:'',
					longitudeEnd:'',
					latitudeStart:'',
					latitudeEnd:'',
					gpsGroup:[],
					itemTest:''
				}
				this.errorMessage = '';
				this.$refs.addTypeForm.resetFields();
			},
			/**
			 * 清除已经选择的基站或者控制类型
			 * @param type {string} 类型
			*/
			clearType(type){
				var vm = this;
				var types = {
					sn:{
						tbRef:'ctableSetting',
						url:"${ctx}/son/access/delAccessTempSN.action",
						resKey:'serialNumber'
					},
					tac:{
						tbRef:'ctableTac',
						url:"${ctx}/son/access/delAccessTempTAC.action",
						resKey:'tac'
					},
					ecgi:{
						tbRef:'ctableEcgi',
						url:"${ctx}/son/access/delAccessTempECGI.action",
						resKey:'ecgi'
					},
					ip:{
						tbRef:'ctableIp',
						url:"${ctx}/son/access/delAccessTempIp.action",
						resKey:'id'
					},
					gps:{
						tbRef:'ctableGps',
						url:"${ctx}/son/access/delAccessTempGPS.action",
						resKey:'id'
					}
				}
				var typeItem = types[type],
					table = vm.$refs[typeItem.tbRef],
					data = table.getChecked(),
					url = typeItem.url,
					result = data.toString().split(",").join(";"),
					params = {
						tempId:vm.tempId,
						[typeItem.resKey]:result
					}
				if(data.length == 0){
					
				}else{
					var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:"warningConfirm",
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						axios.post(url,stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								table.refresh();
							}else{
								vm.$message.error(data["message"])
							}
						}).catch(() => {
							
						})
					})
				}
			},
			//下载模板
			exportTemp(){
				var url = "${ctx}/son/access/importAccessControlTemp.action"
				var params = {tempType : this.showAddType}
				exportByForm(url,params);
			},
			/**
			 * 导出控制方式列表
			 * @param type {string} 类型
			*/
			exportData(type){
				var url = "${ctx}/son/access/exportAccessTempControlCsvFile.action"
				var params = {controlType : type}
				exportByForm(url,params);
			},
			//保存导入后的结果
			saveResult(){
				var vm = this;
				axios.post("${ctx}/son/access/importSaveTemp.action",stringify(this.result_params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.resultVisible = false;
						vm.resultTarget.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				})
			},
			//取消导入后结果页面
			cancelResult(){
				this.resultVisible = false;
				this.resultTarget.refresh();
			},
			//导入数据后结果状态格式化
			statusFmt(row,column,cellValue,index){
				if(cellValue == "legal"){
					return '<%=rb.getString("ChengGong")%>'
				}else{
					return '<%=rb.getString("ShiBai")%>'
				}
			},
			//新建/修改规则提交方法
			submit(){
				var vm = this, yesOrNoChange = false;
				var val = JSON.stringify(vm.oldControlType.sort());
				var orVal = JSON.stringify(vm.checkList.sort());				
				if(val != orVal) yesOrNoChange = true
	   				   
				vm.$refs.ruleForm.validate((valid) => {
					if(valid){
						if(isFormChanged(vm.$refs.ruleForm) || yesOrNoChange == true){
							var params = {
									tempName : vm.ruleForm.tempName,
									status : vm.ruleForm.status,
									controlTacEnable : vm.checkList.includes('tac') ? 1 : 0,
									controlEcgiEnable : vm.checkList.includes('ecgi') ? 1 : 0,
									controlIpEnable : vm.checkList.includes('ip') ? 1 : 0,
									controlGpsEnable : vm.checkList.includes('gps') ? 1 : 0,
									neighborAccess : vm.ruleForm.neighborAccess == true?1:0,
									gpsAccessEnable : vm.ruleForm.gpsAccessEnable == true?1:0
							}
							if(accessVue.operType == 'edit'){
								params.tempId = vm.tempId;
							}else if(accessVue.operType == 'add'){
								params.tempId = "";
							}
							axios.post("${ctx}/son/access/addAccessTemp.action",stringify(params)).then(function(response){
								var data = response.data;
								if(data["success"]){
									vm.$message({
			    						message:"<%=rb.getString("ChengGong")%>",
			    						type:'success',
			    					})
                                    accessVue.$refs.ctableAccessRule.refresh();
			    					accessVue.$refs.slide.hide();
								}else{
									vm.$message.error(data["message"])
								}
							})
						}else{
							vm.$message("<%=rb.getString("WuCanShuBianHua")%>")
						}
					}
				})
			},
			//修改规则时回显数据
			getInfo(){
				var vm = this;
				vm.tempId = accessVue.rowDataRule.id;
				axios.post("${ctx}/son/access/queryAccessTempById.action",stringify({tempId:vm.tempId})).then(function(response){
					var data = response.data;
					vm.params_setting.tempId = accessVue.rowDataRule.id;
					vm.params_tac.tempId = accessVue.rowDataRule.id;
					vm.params_ecgi.tempId = accessVue.rowDataRule.id;
					vm.params_ip.tempId = accessVue.rowDataRule.id;;
					vm.params_gps.tempId = accessVue.rowDataRule.id;
					vm.ruleForm.tempName = data.tempName;
					vm.ruleForm.status = data.status;
					if(data.controlTacEnable=="1"){vm.checkList.push('tac')}
					if(data.controlEcgiEnable=="1"){vm.checkList.push('ecgi')}
					if(data.controlIpEnable=="1"){vm.checkList.push('ip')}
					if(data.controlGpsEnable=="1"){vm.checkList.push('gps')}
					vm.ruleForm.neighborAccess = data.neighborAccess=="1"?true:false;
					vm.ruleForm.gpsAccessEnable = data.gpsAccessEnable=="1"?true:false;
					vm.oldControlType = vm.checkList;
					setTimeout(function(){
						initForm(vm.$refs.ruleForm);
					},500)
				})
				if(accessVue.operType == "view"){
					vm.viewRule = true;
				}
			},
			//取消新建/修改规则页面
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.ruleForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						accessVue.$refs.slide.hide();
					}).catch(() => {
						
					})
				}else{
					accessVue.$refs.slide.hide();
				}
			},
			importControlType(type){
				var typeObj = {
						'sn' : 'Import eNB',
						'tac' : 'Import TAC',
						'ecgi' : 'Import ECGI',
						'ip' : 'Import IP',
						'gps' : 'Import GPS'
					}
					this.showAddType = type;
					this.importTitle = typeObj[type];
					this.importTypeVisible = true;
			},
			closeImportType(){
				this.importTypeVisible = false;
				this.importTypeForm = {
					import_type:'append',
					file_path:'',
				}
				this.errorMessage = '';
				this.$refs.importTypeForm.resetFields();
				$("#uploadForm_access input[name='uploadFile']").val("")
			},
			changeControlType(){
				this.$refs.ruleForm.validateField('typeTest')
			}
		},
		mounted(){
			var vm = this;
			$("#uploadForm_access input[name='uploadFile']").bind("change", function() {
				vm.importTypeForm.file_path = this.value
			});
			eventBus.$off("save-rule").$on("save-rule",vm.submit);
			eventBus.$off("edit-info").$on("edit-info",vm.getInfo);
			eventBus.$off("cancel-rule").$on("cancel-rule",vm.cancel);
		},
		watch:{
			"ruleForm.tacStr":function(){
				this.$refs.ruleForm.validateField('typeTest')
			},
			"ruleForm.ecgiStr":function(){
				this.$refs.ruleForm.validateField('typeTest')
			},
			"ruleForm.ipStr":function(){
				this.$refs.ruleForm.validateField('typeTest')
			},
			"ruleForm.gpsStr":function(){
				this.$refs.ruleForm.validateField('typeTest')
			}
		},
		computed:{
			showType(){
				return this.checkList.length == 0 ? false : true;
			},
			showTac(){
				return this.checkList.includes('tac') ? true : false;
			},
			showEcgi(){
				return this.checkList.includes('ecgi') ? true : false;
			},
			showIp(){
				return this.checkList.includes('ip') ? true : false;
			},
			showGps(){
				return this.checkList.includes('gps') ? true : false;
			}
		}
	})
</script>
