<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.result-info { display: flex; align-items: center; }
	.result-count { align-items: center; justify-content: center; text-align: center; padding: 0 6px; }
	.success-color .el-icon-circle-success::before,
	.status-info .el-icon-circle-success::before { color: #4ED76E; }
	.failure-color .el-icon-circle-close::before,
	.status-info .el-icon-circle-close::before { color: #FF6D59; }
	.failure-color .el-icon-circle-close, .success-color .el-icon-circle-success { font-size: 16px; }
	 i.disabled { opacity: 0.6; }
	.el-tabs .el-tabs__header { border-bottom: 1px solid #E9E9E9; }
	.commonBorder2 { border: 1px solid #E9EDF9; }
	.container, .commonHeight { height: 100%; }
	.commonFlex { display: flex; }
	.commonDisplay { display: inline-block; }
	.commonDisplayBlock { display: block; }
	.commonBorder { border: 1px solid #E9E9E9; }
	.commonBackground { background: #F5F7FE; }
	.commonIconStyle { margin: 10px 15px; width: 30px; height: 30px; background: #E4F1FF; border-radius: 100px; line-height: 30px !important; }
	.commonIconColor:before { color: #7A7992; }
	.commonFontSize14, .commonToolBarBox .el-query .advanceQuery .el-icon { font-size: 14px; }
	.commonFontSize12 { font-size: 12px; }
	.commonColor, .commonToolBarBox .el-query .advanceQuery .el-icon:before, .commonIcon:before, .informationWarp .el-card__header { color: #7A7992; }
	.commonColor2 {  color: #999999; }
	.commonColor3 {  color: #333333; }
	.commonFontWeight { font-weight: bold; }
	.commonRight30 { margin-right: 30px; }
	.commonBorderRadius { border-radius: 10px;}
	.commonToolBarBox { height: 30px; }
	.commonBottom10 { margin-bottom: 10px; }
	.commonRight10 { padding-right: 10px;}
	.commonLeft20 { padding-left: 20px;}
	.commonContent { justify-content: space-between; }
	.addBtn {  right: 0px !important;}
	.eNBTableTitle { line-height: 30px;}
	.commonToolBarBox .el-query{ right: 56px; position: absolute !important;}
	.commonToolBarBox .el-query .advanceQuery { height: 28px; border-radius: 8px; }
	.commonToolBarBox .el-query .advanceQuery .el-input.el-input--small{ width: 280px; }
	.commonToolBarBox .el-query .advanceQuery .el-input.el-input--small .el-input__inner { height: 26px; line-height: 26px; }
	.el-table th { background: #F9F9F9; }
	.el-table td { color: #666666; }
	.el-table--border th { border-right: 1px solid #E9E9E9; }
	.el-table th>.cell { color: #999999; font-weight: normal; }
	.radioBox { background-color: #FFFFFF !important; }
	.radioBox .el-radio__label { font-weight: 600; color: #7A7992; background: #FFFFFF; font-size: 16px;}
	.radioBox .el-radio__input { display: none; }
	.el-icon-status-disable:before { color: #B8C3D9; }
	.el-icon-status-enable:before { color: #4ED76E; }
	.statusText { font-size: 12px; color: #666666; }
	.el-progress-bar { width: 85px; }
	.el-progress__text { display: none; }

	.commonWidth166 .el-input{ width: 150px; }
	.informationWarp { width: calc(100% - 20px) !important; height: 218px !important; margin: 0 8px; bottom: 50px !important; left: 2px !important; }
	.informationWarp .el-card { border-radius: 0 0 10px 10px; }
	.informationWarp .el-card__header span:last-child { right: 4px !important; }
	.informationWarp .el-card__header .el-icon-close {  font-size: 14px; }
	.el-switch.is-checked .el-switch__core { border-color: #4D84FF !important; background-color: #4D84FF !important; }
	.addSlide { width: 100% !important; height: 100% !important; border-radius: 10px; }
	.addSlide .el-card__header { height: 40px !important; line-height: 40px !important; }
	.tabTitleIcon { padding: 0 10px 0 0; }
	.tabTitleIcon:before { font-size: 16px; }
	.el-tabs__item.is-active .tabTitleIcon:before { color: #4D84FF; }
	.queryGroup { height: 26px; border-radius: 8px; }
	.queryGroup .el-input { width: 280px; }
	.queryGroup .el-input__inner { width: 280px; height: 26px; line-height: 26px; padding: 0; }
	.queryGroup .el-icon-common-search { font-size: 14px; }
	.queryGroup .el-icon-common-search:before { color: #7A7992; }
	.el-date-editor .el-range-input { font-size: 12px; }
	.el-date-editor .el-range__close-icon { margin-top: -12px; } 

	.addBtnStyle { top: 0; margin-right: 0px !important; }
	.el-form-item { margin-bottom: 26px; }
	
	.leftWarp {flex: 1; height: 100%; position: relative; overflow: hidden; flex-direction: column; border-radius: 10px; }
	.rightBox { flex: 0 322px; height: calc(100% - 2px);  margin-left: 10px; border-radius: 10px; overflow: hidden; }
	.commonFormFotter { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	.commonBackgroundWhite { background: #FFFFFF; }
	.commonTitle { height: 48px; line-height: 48px; }
	.rightHeaderBox .addTitle {  padding: 0; }
	.rightTextBox { padding: 10px 0; }
	.commonBorderBottom { border-bottom: 1px solid #E9EDF9; }
	.commonBorderTop { border-top: 1px solid #E9EDF9; }
	.rightContent { padding: 16px 0 0; }
	.rightContent .el-input{ width: 282px } 
	.rightContent .el-form-item__label { color: #666666; }
	.rightContent .el-input__inner{ height: 30px; line-height: 30px; }
	.rightContent .el-input-group__append { padding: 0; background-color: #FFFFFF; }
	.rightContent .el-input-group__append .el-icon { font-size: 14px; }
	.rightContent .el-input-group__append .el-icon:before { color: #7A7992; }
	.ipErrorTip { color: #FA5555; font-size: 12px; }
	.resultBox { border-radius: 4px; background-color: #FFFFFF; margin-top: 5px; width: 280px; max-height: 122px; padding: 5px 0; overflow: auto;}
	.resultBox .el-form-item { margin-right: 0 !important;}
	.form-suffix { position: relative; margin: 3px 0 0 20px; padding: 0; width: 240px; border: none; background: #fff; }
	.form-suffix:hover { background: #F4F9FF; border-radius: 100px; }
	.form-suffix .deleteCommon { display: none; position: absolute; right: 0; top: 8px; }
	.form-suffix:hover .deleteCommon { display: inline-block !important; }
	.form-suffix .text { padding-left: 10px; color: #666666; }
	.form-suffix .ipTextWarp { font-size: 14px; }
	.operBtn { width: 26px; height: 26px; border: 1px solid #D7D7E6; border-radius: 8px; margin-top: 1px; text-align: center; }
	.operBtn .el-icon { line-height: 26px; }
	.operBtn .el-icon:before, .importIcon:before { color: #7A7992; }
	.container .group { padding-left: 26px; }
	.group-title { color: #7A7992; }
	.mainContent .el-form-item { margin-bottom: 22px; }
	.mainContent .el-form-item__label { line-height: 28px; } 
	.AddTitle { padding: 0 30px; }
	.el-dialog__footer { height: 48px; }
	.commonLeft10 { margin-left: 10px; }
	.licenseImportBox { margin: 0 10px; width: 26px; height: 26px; border: 1px solid #D7D7E6; text-align: center; border-radius: 8px; }
	.licenseImportBox i { font-size: 12px; line-height: 26px; }
	.el-input-group__prepend, .el-input-group__append { font-size: 14px; padding: 0 6px;}
	.el-dialog__footer { border-top: 1px solid #E9EDF9; }
	.productClassBox { margin: 0 !important;}
	.productClassBox .el-input { width: 60px; }
	.productClassBox .el-input__inner { height: 26px; line-height: 26px; border-left: none; padding: 0 6px; }
	#modelPage .importFieleItem .el-input__suffix { margin-top: 6px; }
	.importFieleItem .el-form-item__error { margin-top: -8px; }
	.productTypeInfoBox { padding: 0px 18px; float: left; line-height: 28px; height: 28px; border-radius: 30px; background: #F6F7FB; border: 1px solid #7A7992; margin: 0 10px 10px 0; }
	.el-tooltip__popper.is-dark { margin: 0 30px 0 50px; }
</style> 

<div id="modelPage" class="container" style='position: relative;'>
	<!-- Model Name-->
	<el-radio-group v-model='activeTab' @change='clickTab' class='radioBox' style='position: absolute; top: 21px; left: 20px; z-index: 999;'>
		<el-radio label="name"><span v-if='false' class='el-icon el-icon-menu-advance'></span> <%=rb.getString("SheBeiLeiXing") %></el-radio>
		<el-radio label="type" style='margin-left: 50px;'> <span v-if='false' class='el-icon el-icon-menu-advance'></span> <%=rb.getString("ShuJuMoXing")%></el-radio>	
	</el-radio-group>		
	<div class='commonFlex' style='width: 100%; height: 100%;' v-show='activeTab == "name"'>
		<div class='leftWarp'>
			<!-- 操作按钮 -->
			<div class="operations addBtn" style='top: 16px;'>
				<div class="circleIcon placeholder-bt addBtnStyle" placeholder="<%=rb.getString("XinZeng")%>">		
					<span class="el-icon-circle-add el-icon" @click="addProductModelClick"></span>
				</div>
			</div>
			<el-ctable height='99.5%' ref="modelNameTable" class='commonBorderRadius commonBorder ' style='margin-top: 2px;' :time='6' id='modelNameList'
				:url="modelNameUrl" :pagination="true" 
				:query-params="modelNameParam"
				:row-key="'id'">
				<div slot="toolbar">
					<div class='commonFlex commonToolBarBox' style='position: relative;'>
						<el-query type="normal" @query="queryModelName" placeholder="<%=rb.getString("SheBeiXingHaoMing") %> / <%=rb.getString("ChanPinLeiXing")%> / Param_Model"></el-query>
					</div>
				</div>
				<el-table-column width="40">
					<template slot-scope="scope">
						<div class="el-icon el-icon-operation-more" @click="proModelOptClick(scope.row, event)" v-clickoutside="modelHanderClose"></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("SheBeiXingHaoMing")%>" prop="modelName" show-overflow-tooltip="true" sortable></el-table-column>
				<el-table-column label="<%=rb.getString("ChanPinLeiXing")%>" prop="productType" show-overflow-tooltip="true" sortable></el-table-column>
				<el-table-column label="Param_Model" prop="paramModel" show-overflow-tooltip="true" sortable></el-table-column>
				<el-table-column label="<%=rb.getString("ChanPinMingCheng") %>" prop="productName" show-overflow-tooltip="true" sortable></el-table-column>	
			</el-ctable>
			<el-cmenu ref="productModelMenu" :data="productModelMenus" @click="menuClick"></el-cmenu>
		</div>
		
		<!-- new model name-->
	   	<div class='rightBox commonBorder2 commonBackgroundWhite ' style='position: relative;' v-show='addProductModelShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
				<span class='AddTitle commonTextColor commonFontSize14 commonFontWeight'>{{ModelNameTitle}}</span>
				<span class='' @click='addProductModelCancel'><i class='el-icon el-icon-circle-close'></i></span>
			</div>
	   		<div class='rightContent' style='padding: 26px 20px; height: calc(100% - 100px);overflow-x: scroll;'> 
	   			<el-form :model="confirmForm" ref="confirmForm" label-position="top" :rules="formRule">
					<el-form-item style='margin-bottom: 22px;' label='<%=rb.getString("MoKuaiXingHao") %>'>
						<el-form-item prop='modelName'>
							<el-input v-model='confirmForm.modelName' class="input-with-select" maxlength='100' :disabled='curType == "edit"'>
								<el-select v-model="confirmForm.productTypeClass" slot="append" style='width: 60px;' class='productClassBox' v-if='curType == "add"'>
					                   <el-option v-for="item in productClass" :label="item.name" :value="item.value"></el-option>
					             </el-select>
								 <i slot="append" class="el-icon el-icon-plus" @click="addProductModelBtn" style='width: 34px; text-align: center;' v-if='curType == "add"'></i>
							</el-input>						
						</el-form-item>
		            </el-form-item> 
		            <el-form-item label="Product Class" prop="productClass" class='inputCommon'>
		                <el-input v-model="confirmForm.productClass" ></el-input>
		            </el-form-item>
		            <el-form-item label="<%=rb.getString("ChanPinLeiXing")%>" prop="productType" class='inputCommon'>
		                <el-input v-model="confirmForm.productType"></el-input>
		            </el-form-item>
		            
					<el-form-item label="<%=rb.getString("ChanPinLeiXingLieBiao")%>" style='margin-bottom: 18px;'>
						<el-ctable height="35%" ref="productTypeTable" class='commonBorderRadius commonBorder commonBottom10' style='margin-top: 2px;'
							:row-key="'productType'" :time='6' id='productTypeList'
							:url="productTypeListUrl" :pagination="false" :rownumber=false 
							@row-click="rowClickChange">
							<el-table-column width="50">
								<div slot-scope="scope" style="margin: 0 auto;">
									<el-radio v-model="confirmForm.curSelectProductType" :label="scope.row.productType"><span></span></el-radio>
								</div>
							</el-table-column> 
							<el-table-column label="<%=rb.getString("ChanPinLeiXing")%>" prop="productType" show-overflow-tooltip="true" sortable></el-table-column>
						</el-ctable>
			            <p class='ipErrorTip' style='margin-top: -12px;'>{{productModelMessage}}</p>
					</el-form-item>
					<el-form-item label="Param_Model" prop="paramModel" placeholder="<%=rb.getString("QingXuanZe") %>" class='selectCommon' v-show='productTypeSelectShow'>
		                <el-select v-model="confirmForm.paramModel">
		                    <el-option v-for="item in paramModelList" :label="item.name" :value="item.value"></el-option>
		                </el-select>
		            </el-form-item>
		            <el-form-item label="Param_Model" prop="paramModelInput" class='inputCommon' v-show='productTypeInputShow'>
		                <el-input v-model="confirmForm.paramModelInput" disabled></el-input>
		            </el-form-item>
					<el-form-item label="<%=rb.getString("ChanPinMingCheng") %>" prop="productName" class='inputCommon'>
		                <el-input v-model="confirmForm.productName"></el-input>
		            </el-form-item>
				</el-form>   				   				
	   		</div>
			<div class='commonFlex commonBorderTop commonFormFotter'>
				<div style='padding: 10px 30px 0;'>
					<el-button type="primary" size="mini" @click='addProductModelSubmit'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='addProductModelCancel'><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
	   	</div>
	</div>

	<!-- 数据模型-->
	<div class='commonFlex ' style='width: calc(100% - 2px); height: calc(100% - 2px);' v-show='activeTab == "type"'>
		<div class='leftWarp commonBackgroundWhite commonBorderRadius commonBorder'>
			<!--Param_model list -->
			<div class='commonFlex' style='height: 100%;'>
				<div style='position: relative;  width: 220px; height: 100%;'>
					<div class='commonFlex commonContent' style='margin-top: 52px; border-top: 1px solid #E9EDF9; border-right: 1px solid #E9EDF9'>
						<div class="splitTitle commonFontSize14 commonColor" style="padding:10px; margin-right: 32px;">Param_model</div>
						<div class='licenseImportBox' @click='importParamModelBtn' style='margin-top: 5px;' v-if="false"><i class='el-icon el-icon-operation-import importIcon'></i></div>
					</div>
					<el-ctable ref="paramModelTable" :url="paramModelUrl" id='paramModelList' style='border-right: 1px solid #E9EDF9;  height: calc(100% - 91px);'
						:show-pager="false" :show-header=false highlight-current-row="true" 
						:row-key="'param_model'" :page-size="templatePageSize" :page-list="templatePageList" :pagination="true" :rownumber=false highlight-current-row="true"
						@load-success="tableLoadSuccess" 
						@row-click="paramModelClick">
						<el-table-column label="<%=rb.getString("ChanPinLeiXing")%>" prop="param_model" show-overflow-tooltip="true"></el-table-column>
						<el-table-column width="40">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="groupOpClick(scope.row, event)" v-clickoutside="modelHanderClose"></div>
							</template>
						</el-table-column>
					</el-ctable>
					<el-cmenu ref="menuGroup" :data="menusGroup" @click="clickMenu"></el-cmenu>
				</div>
				<div style='flex-grow: 1; overflow: auto; margin-top: 52px; border-top: 1px solid #E9EDF9; '>
					<el-ctable height="100%" ref="paramList" :time='6' id='paramListTable' 
						:row-key="'id'" :pagination="true" 
						:url="paramListURL"
						:query-params="parmParams"
						@row-click="rowClick">
						<div slot="toolbar">
							<div class='commonFlex commonToolBarBox' style='position: relative;'>
								<span v-if="paramModelClickData.length !== 0 " class='commonTextColor commonFontSize14' style='margin-left: 16px;'>{{paramModelClickData.param_model}}</span>
								<div class='commonToolBarBox'>
									<el-query type="normal" @query="queryParamPath" placeholder='<%=rb.getString("GongYouPath") %> / <%=rb.getString("SiYouPath")%>' style="right: 15px;"></el-query>
								</div>														
								<div class="operations" style='top: 3px; ' v-if="false">
									<div class="circleIcon placeholder-bt addBtnStyle" placeholder="<%=rb.getString("XinZeng")%>">		
										<span class="el-icon-circle-add el-icon" @click="addParamBtn"></span>
									</div>
								</div>							
							</div>
						</div>
						<el-table-column width="40">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="paramListClick(scope.row, event)" v-clickoutside="modelHanderClose"></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("GongYouPath")%>" prop="standardPath" show-overflow-tooltip="true" width="200"></el-table-column>
						<el-table-column label="<%=rb.getString("SiYouPath")%>" prop="privatePath" show-overflow-tooltip="true" width="200"></el-table-column>
						<el-table-column label="<%=rb.getString("DuXieQuanXian")%>" prop="writable" show-overflow-tooltip="true" width="200">
							<template slot-scope="scope" class="status-info">
								<div v-if="scope.row.writable=='R'"><%=rb.getString("ZhiDu")%></div>
								<div v-if="scope.row.writable=='W'"><%=rb.getString("ZhiXie")%></div>
								<div v-if="scope.row.writable=='RW'"><%=rb.getString("DuXie")%></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("ShuJuLeiXing")%>" prop="dataType" show-overflow-tooltip="true" width="200">
							<template slot-scope="scope" class="status-info">
								<div v-if="scope.row.dataType=='boolean'">Boolean</div>
								<div v-if="scope.row.dataType=='dateTime'">DateTime</div>
								<div v-if="scope.row.dataType=='int'">Int</div>
								<div v-if="scope.row.dataType=='unsignedInt'">UnsignedInt</div>
								<div v-if="scope.row.dataType=='string'">String</div>
							</template>
						</el-table-column>
						<!-- rangeType: num-长度范围， enum-枚举范围；  数据类型：minLength,maxLength 归属 string;  max,min 归属 init-->
						<el-table-column  label="<%=rb.getString("QuZhiFanWeiBanMian")%>" prop="enumValueOfUser" show-overflow-tooltip="true" width="220">
							<template slot-scope="scope" class="status-info">
								<div v-if='scope.row.rangeType == "num" && scope.row.dataType == "string"'>{{ scope.row.minLength}},{{scope.row.maxLength}}</div>
								<div v-if='scope.row.rangeType == "num" && (scope.row.dataType == "int" || scope.row.dataType == "unsignedInt")'>{{scope.row.min}},{{ scope.row.max}}</div>
								<div v-if='scope.row.rangeType == "enum"'>{{ scope.row.enumValueOfUser}}</div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("QuZhiFanWeiMingLing")%>" prop="enumValueOfDevice" show-overflow-tooltip="true" width="220"></el-table-column>
						<el-table-column label="<%=rb.getString("DongTaiShengXiao")%>" prop="isDynamic" show-overflow-tooltip="true" width="140">
							<template slot-scope="scope" class="status-info">
								<div v-if="scope.row.isDynamic=='0'"><%=rb.getString("JingTaiShengXiao")%></div>
								<div v-if="scope.row.isDynamic=='1'"><%=rb.getString("DongTaiShengXiao")%></div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("EnbBeiZhu")%>" prop="privateName" show-overflow-tooltip="true" width="200"></el-table-column>								
					</el-ctable>
					<el-cmenu ref="menuParamList" :data="menusParamList" @click="clickParamListMenu"></el-cmenu>
				</div>
			</div>
		</div>
		
		<!-- importParam-->
	   	<div class='rightBox commonBorder2 commonBackgroundWhite ' style='position: relative;' v-show='importParamShow'>
	   		<div class='commonBorderBottom'>
		   		<div class='commonFlex commonContent commonTitle rightHeaderBox ' style='padding: 0 20px;'>
					<span class='AddTitle commonTextColor commonFontSize14 commonFontWeight'><%=rb.getString("DaoRuCanShuBiao")%></span>
					<span class='' @click='importParamCancel'><i class='el-icon el-icon-circle-close'></i></span>
				</div>
				<span class='commonColor2 commonFontSize12' style='margin: -2px 20px 10px; display: block;'><%=rb.getString("ShuJuMoXingYiCunZai")%></span>
	   		</div>
	   		<div class='rightContent' style='padding: 26px 20px; height: calc(100% - 100px);overflow-x: scroll;'> 
	   			<el-form label-position="top" ref="importParamForm" :model='importParamForm' :rules='importRules'>          
                    <el-form-item label="Param_Model" prop="paramModel" class='selectCommon'>
		                <el-input v-model="importParamForm.paramModel" maxlength='100'></el-input>
		                <p v-if='false'>Length: 100</p>
		            </el-form-item>  
                  	<el-form-item label="File" prop="fileName" class='importFieleItem'>
	                  	<el-upload :before-upload='beforeUpload' 
	                  	:on-success='checkFile' 
	                  	:on-change="fileChange"  
	                  	:show-file-list=false ref="upload"  accept=".xlsx"
						:action="importParamForm.uploadFileUrl" 
						:data="fileParams" 
						name="uploadFile" :auto-upload="false">
							<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w270">
								<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
							</el-input>								
							<a slot="trigger" ref="file_up"></a>
						</el-upload>
						<div class='commonContent commonFlex'>
							<span class='commonColor2 commonFontSize12'>Only .xlsx is supported</span>
							<span @click="exportTemplate" class='commonColor2 commonFontSize12'><i style='vertical-align:center' class='el-icon el-icon-common-download commonColor2'></i><%=rb.getString("DaoChuMuBan")%></span>
						</div>		
                    </el-form-item> 
                    <el-form-item style='margin-bottom: 16px;' label='<%=rb.getString("ChanPinLeiXing") %>'>
						<el-form-item prop='productModelStr'>
							<el-input v-model='importParamForm.productModelStr'>
								 <span slot="append" class="el-icon el-icon-plus" @click="addProductTypeBtn" style='width: 30px; text-align: center;'></span>
							</el-input>						
						</el-form-item>
						<div class='commonBorder2 resultBox' v-show='importParamForm.productTypeList.length > 0'>
							<el-form-item class='suffixItem' v-for='(domain,index) in importParamForm.productTypeList'>
								<div class='form-suffix'>
									<span class='text'>{{domain}}</span>
									<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteCommon' @click.prevent='removeProductType(domain)'></span>
								</div>
							</el-form-item>
							<el-form-item prop='itemTest'>
								<el-input v-model='importParamForm.itemTest' v-show=false></el-input>
							</el-form-item>
						</div>             
						<p class='ipErrorTip'>{{importProductTypeErrorMessage}}</p>
		            </el-form-item> 
					<el-form-item label="<%=rb.getString("ChanPinLeiXingLieBiao")%>" style='margin-bottom: 18px;'>
						<el-ctable ref="productTypeTable"  class="commonBorder2" :url="productTypeListUrl"
			            	:pagination="false" :rownumber=false :row-key="'productType'" :time='6' id='productTypeList'
			            	@selection-change='productModelBatchSelect' height="210px">	                          
			                <el-table-column type="selection" :reserve-selection="true"></el-table-column>
							<el-table-column label="<%=rb.getString("ChanPinLeiXing") %>" prop="productType"></el-table-column> 	                          
			            </el-ctable> 
			            <p class='ipErrorTip'>{{importProductTypeMessage}}</p>
					</el-form-item>     
	            </el-form>  				   				
	   		</div>
			<div class='commonFlex commonBorderTop commonFormFotter'>
				<div style='padding: 10px 30px 0;'>
					<el-button type="primary" size="mini" @click='importParamSubmit'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='importParamCancel'><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
	   	</div>
	</div>

	<!-- Add Config Slider -->
	<el-slide ref="slider" title="<%=rb.getString("XinZeng")%>" 
		:url="slideURL" 
		:title="slideTitle" 
		:header='slideHeader' 
		:footer="footerShow" class='addSlide'
		@ok="addConfig"
		@cancel="closeConfig">
	</el-slide>
	
	<el-dialog title="<%=rb.getString("QueRen")%>" id="delDialog" :visible.sync="delDialog" top="30vh" ref="delDialog" width="460" :close-on-click-modal="false" @close='cancelDel' append-to-body>
		<div v-show='curDelTtype == "modelNameDel"'>
			<div style="margin-bottom:10px; margin-top: 15px;"><%=rb.getString("ShiFouQueRenShanChuGuanXiBiao")%></div>
			<span class='commonColor2 commonFontSize12' style="margin-bottom:20px;"><%=rb.getString("ShanChuZaoChengSheBeiCanShuShiBai")%></span>
		</div>
		<div v-show='curDelTtype == "paramModelDel"'>
			<div style="margin-bottom:10px; margin-top: 15px;"><%=rb.getString("QueRenShanChuShuJuMoXing")%></div>
			<span class='commonColor2 commonFontSize12' style="margin-bottom:20px;"><%=rb.getString("ShanChuShuJuMoXingTiShi")%></span>
		</div>
		<div v-show='curDelTtype == "paramListDel"'>
			<div style="margin-bottom:10px; margin-top: 15px;"><%=rb.getString("QueRenShanChuCanShuBiao")%></div>
			<span class='commonColor2 commonFontSize12' style="margin-bottom:20px;"><%=rb.getString("ShanChuShuJuMoXingTiShi")%></span>
		</div>
		<div slot="footer">
			<div style='float: right; margin-top: 12px; '>
				<el-button type="primary" @click="delSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="cancelDel"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-dialog>
	
	<!-- param_model-info -->
	<el-dialog title="<%=rb.getString("XinXi")%>" id="infoDialog" :visible.sync="paramModelInfoDialog" top="30vh" ref="infoDialog" width="560" 
	:close-on-click-modal="false" @close='cancelInfo' append-to-body>
		<div style='padding-bottom: 30px;'>
			<div class='commonFlex'>
				<span class='commonColor2 commonFontSize12'>Param_Model</span>
				<span style='margin-left: 20px;'>{{paramModelInfo}}</span>
			</div>
			<div class='commonFlex' style='padding-top: 12px;'>
				<p class='commonColor2 commonFontSize12' style='width: 100px; float: left;'><%=rb.getString("ChanPinLeiXing")%></p>
				<div style='width: 400px; max-height: 200px; overflow: auto;'>
					<div class='productTypeInfoBox' v-for='(domain,index) in productTypeInfo'>
						<p class='text'>{{domain}}</p>
					</div>
				</div>
			</div>
		</div>
		<div slot="footer">
			<div style='float: right; margin-top: 12px; '>
				<el-button type="primary" @click="cancelInfo"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="cancelInfo"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-dialog>
	<!-- param_model-已存在 -->
	<el-dialog title="<%=rb.getString("QueRen")%>" id="infoDialog" :visible.sync="importParamModelExistsDialog" top="30vh" ref="infoDialog" width="560" 
	:close-on-click-modal="false" @close='cancelImportParamModelExists' append-to-body>
		<div style='padding-bottom: 30px;'>
			<div style="margin-bottom:10px; margin-top: 15px;">Param_Model: {{existsParamModel}} <%=rb.getString("YiCunZai")%> <%=rb.getString("QueRenYaoTiHuan")%></div>
			<span class='commonColor2 commonFontSize12' style="margin-bottom:20px;"><%=rb.getString("TiHuanDuiYingDeCanShuBiao")%></span>
		</div>
		<div slot="footer">
			<div style='float: right; margin-top: 12px; '>
				<el-button type="primary" @click="confirmImportParamModelExists"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="cancelImportParamModelExists"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-dialog>
</div>

<script>
	var modelPageVue = new Vue({
		el: '#modelPage',
		data() {
			var vm = this,
				validateProductModel = function(rule,value,callback){
					if(value === '' || value === null || value === undefined) {
						callback(new Error('<%=rb.getString("BiTian")%>'));
					}else{
						callback();
					}
				},
				validateFileName = function(rule,value,callback) {
	            	value = vm.fileName;     		
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else if(!fileFormatMatch(value,"xlsx")){
                        callback(new Error('<%=rb.getString("DaoRuWenJianLeiXingXLSX")%>'))
                    }else {
						callback();
					}
				},
				validateModelName = function(rule,value,callback){
					if(value === '' || value === null || value === undefined) {
						callback(new Error('<%=rb.getString("QingShuRuXingHaoMingCheng")%>'));
					}else{
						callback();
					}
				},
				validateParamModel = function(rule,value,callback){
					if(vm.productTypeSelectShow ==  false){
						callback();
					}else{
						if(value === '' || value === null || value === undefined) {
							callback(new Error('<%=rb.getString("QingShuRuXingHaoMingCheng")%>'));
						}else{
							callback();
						}
					}
				};
			return {				
				activeTab: 'name',
				modelNameUrl: '${ctx}/dataModel/enb/productModelProductNameRela/getList.action',
				modelNameParam: {
					search_text: '',
					timeZone: timeZone,
					like_fields: 'product_name,model_type,product_type',
					page: 1,
					rows: 50
				},
				productTypeList: [],				
				slideHeader: false,
				footerShow: false,
				currentRow: [],
				productCpeList: [],
				slideTitle: '',
                slideURL: '',
				//new product model
				productModelMenus: [],
				productTypeMenus: [],
                addProductModelShow: false,
            	productModelMessage: '',
				confirmForm:{
					id: '', 
					modelName: '',
					productType: '',
					productClass: '',
					productTypeClass: '',
					paramModel: '',
					paramModelInput: '',
					productName: '',
					curSelectProductType: '',
				},
				formRule: {
					modelName: [{validator: validateModelName}],
					productClass: [{validator: validateProductModel}],
					productName: [{validator: validateProductModel}],
					paramModel: [{validator: validateParamModel}],
            	},
            	oldProductClass: '',
            	productTypeListUrl: '${ctx}/dataModel/enb/productModelProductNameRela/getProductTypeList.action',

				pageSize: 50,
				rowData: [],
				curType: '',
				ModelNameTitle: '',
				isReadOnly: false,
				delDialog: false,
				//add product type				
				productClass: [
					{ name: '', value: '' },
					{ name: 'CA', value: 'CA' },
					{ name: 'DC', value: 'DC' },
					{ name: 'SC', value: 'SC' },
					{ name: 'TC', value: 'TC' },
				],
				paramModelList: [],
				productTypeSelectShow: false,
				productTypeInputShow: false,
				addProTypeShow: false,
            	//param-model
    			menusGroup: [],
    			templatePageSize:50,
    			templatePageList:[50,100,200],
    			paramModelClickData: [],
    			parmParams: {
    				paramModel: '',
    				search_text: '',
					timeZone: timeZone,
					like_fields: 'standard_path,private_path',
    			},
    			curDelTtype: '',
    			paramModelUrl: '${ctx}/dataModel/enb/standardPrivateRela/getParamModelListByPage.action',

 				// data model: import param_model
 				importParamShow: false,
 				importParamForm: {
 	                uploadFileUrl: '',
 	                paramModel:'',
 	                productModelStr: '',
					productTypeList: [],
					itemTest: ''          
 	           	},	         	          
 	            fileParams:{},              
 	            fileName:'',	            					
 				showFileTip:false,
 				fileList:[],
 				filePath:'',
 				importRules: {
 					paramModel: [{validator: validateModelName}],
 					fileName: [{validator: validateFileName}]                   
 	            },
 	           	mergeProductTypelList: [],
 	            curSelectProductModelData: [],
 	            importProductTypeErrorMessage: '',
				importProductTypeMessage: '',
 	            paramModelRowData: [],
 	           	paramListURL: '${ctx}/dataModel/enb/standardPrivateRela/getList.action',
 	          	menusParamList: [],
 	          	paramListClickData: [],
 	          	paramModelInfoDialog: false,
 	          	paramModelInfo: '',
 	          	productTypeInfo: [],
 	          	importParamModelExistsDialog: false,
 	          	existsParamModel: ''
			}
		},
		watch: {
			curSelectProductModelData(row){
				if(row.length != 0){
					this.importProductTypeMessage = '';
				}    			
			},

    		//手动输入
    		'confirmForm.productType':function(newVal){
    			var vm = this;
				if(newVal === '' || newVal === null || newVal === undefined){
					vm.productTypeSelectShow = false;
				}else{
					vm.productTypeSelectShow = true;
					vm.productTypeInputShow = false;
					vm.confirmForm.curSelectProductType = '';
					vm.confirmForm.paramModelInput = '';
				}
			},
			//列表选中
			'confirmForm.curSelectProductType':function(newVal){
    			var vm = this;
    			
				if(newVal === '' || newVal === null || newVal === undefined){
					vm.productTypeInputShow = false;
				}else{
					vm.productTypeInputShow = true;
					vm.productTypeSelectShow = false;
					vm.confirmForm.productType = '';
					vm.confirmForm.paramModel = '';
				}
			},
		},

		methods: {
			//-----------------------------------------------------add product model
			//右上角 add model name
			addProductModelClick(){
				var vm = this;
				vm.curType = 'add';
				vm.isReadOnly = false;
				vm.ModelNameTitle = 'New Model Name';
				vm.importParamShow = false;
				vm.addProductModelShow = true;
				vm.$refs.confirmForm.resetFields();
				vm.productModelMessage = '';
				vm.confirmForm.productType = '';
				vm.confirmForm.paramModel = '';
			
				vm.confirmForm.curSelectProductType = '';
				vm.confirmForm.paramModelInput = '';
				vm.productTypeSelectShow = false;
				vm.productTypeInputShow = false;
				vm.commonParamModelList();
			},
			// param_model
			commonParamModelList(){
				var vm = this;
				axios.post("${ctx}/dataModel/enb/standardPrivateRela/getParamModelListByPage.action",stringify({
                   // page: 1,
                   // rows: 50
                })).then(function(res){
					var data = res.data.rows;
					vm.paramModelList = (data||[]).map(function(item){
			   			if (item){
			   				return {name: item.param_model, value: item.param_model}
			   			}
			   		})
				});
			},
			// 导入 导出 end

			//手动添加 product type
			addProductModelBtn(){
				var vm = this, curModelName = vm.confirmForm.modelName;
				if(curModelName === '' || curModelName === null || curModelName === undefined){
					vm.confirmForm.productClass = '';
				}else{
					vm.confirmForm.productClass = 'FAP/'+curModelName+'/' + vm.confirmForm.productTypeClass;
				}
			},
			//单行选中 product type,映射 param_model
			rowClickChange(row,old){
	    		var vm = this;
				if(row) {
					vm.confirmForm.curSelectProductType = row.productType;
					vm.confirmForm.paramModelInput = row.paramModel;	
				}
	    	},
			//添加保存
			addProductModelSubmit(){
        		var vm = this, addflag = false, url = '';
        		var newVal = vm.confirmForm.productType, curSelectType = vm.confirmForm.curSelectProductType; 

    			if((newVal === '' || newVal === null || newVal === undefined) && (curSelectType === '' || curSelectType === null || curSelectType === undefined)){
        			vm.productModelMessage = '<%=rb.getString("ChanPinLeiXingZhiShaoTianJiaYiGe")%>';
        			addflag = true;
        		}else{        		
        			addflag = false;
        		}
    			var params = {
              			modelName: vm.confirmForm.modelName.split(','),
              			productClass: vm.confirmForm.productClass,
              			productName: vm.confirmForm.productName
              		};
        		if(curSelectType === '' || curSelectType === null || curSelectType === undefined){
    				params.productType = vm.confirmForm.productType;
    				params.paramModel = vm.confirmForm.paramModel;
    			}else{
    				params.productType = vm.confirmForm.curSelectProductType;
    				params.paramModel = vm.confirmForm.paramModelInput;
    			}
        		if(vm.curType == 'add'){
        			
           			url = "${ctx}/dataModel/enb/productModelProductNameRela/addInfo.action";
        		}else{
        			params.id = vm.rowData.id;
        			//新增的参数为，修改前的 productClass;
        			params.oldProductClass = vm.oldProductClass;
        			url = "${ctx}/dataModel/enb/productModelProductNameRela/modifyInfo.action";
        		}

        		var saveParams = JSON.stringify(params);
        		vm.$refs.confirmForm.validate(function(valid){
					if(valid && addflag == false){
						axios.post(url,saveParams,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
							var data = response.data;
							var message = '<%=rb.getString("ChengGong")%>';
							if(data["success"]){
								vm.$message({
		    						message:message,
		    						type:'success',
		    					})
                                vm.$refs.modelNameTable.refresh();
		    					vm.addProductModelShow = false;
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				});
			},
			//添加取消
			addProductModelCancel(){
				var vm = this;
				vm.addProductModelShow = false;
        		vm.$refs.confirmForm.resetFields();
        		vm.confirmForm.productTypeList = [];
			},
			
			//rowClick
			rowClick(row) {
				this.currentRow = row;
			},
			//-------------------------------------------------------------------table operation
			clickTab(){
				var vm = this;
				vm.addProductModelShow = false;
				vm.addProTypeShow = false;
				vm.importParamShow = false;
				vm.addProductModelShow = false;
			},
			//device type 表格操作
			proModelOptClick(row, ev) {
				var vm = this, proModelDelFlag = false;
				
				vm.rowData = row;
				
				if(row.is_build_in == '1'){
					proModelDelFlag = true;
    			}else{
    				proModelDelFlag = false;
    			}
				
				vm.productModelMenus= [
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'modify', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del', row: row, disable: proModelDelFlag}
				];
				
		    	vm.$nextTick(function(){
		    		document.body.click();
		    		if(vm.activeTab == 'name'){
		    			vm.$refs.productModelMenu.show(ev);
		    		}else{
		    			vm.$refs.productTypeMenu.show(ev);
		    		}
		    	});
			},
			// menu click
			menuClick(ev) {
				var vm = this,
					codes = {
						modify: vm.modifyModel,
						del: vm.deleteModel
					};
				if(codes[ev.code]){
					codes[ev.code](vm.rowData)
				}
			},
			
			// modify
			modifyModel(row){
				var vm = this;
				vm.curType = 'edit';
				vm.isReadOnly = true;
				vm.commonParamModelList();
				vm.productTypeSelectShow = true;
				vm.productTypeInputShow = false;
				if(row){
					vm.confirmForm.modelName = row.modelName;
					vm.confirmForm.productClass = row.productClass;
					vm.confirmForm.productName = row.productName;
					//如何回显 product type 和 param_model,按照input 补充值
					vm.confirmForm.productType = row.productType;
					vm.confirmForm.paramModel = row.paramModel;
					vm.oldProductClass = row.productClass;
				}
				vm.productModelMessage = '';
				vm.ModelNameTitle = 'Modify Model Name';
				vm.addProductModelShow = true;
			},
			//delete
			deleteModel(row) {
				var vm = this;
				vm.delDialog = true;
				vm.addProductModelShow = false;
				vm.curDelTtype = 'modelNameDel';
			},	

			//-------------------------------------------数据模型
			//param list add
			addParamBtn(){
				var vm = this,
				url = '${ctx}/dataModel/enb/standardPrivateRela/goAddDataModelJSPPage.action';
				vm.importParamShow = false;
				vm.slideTitle = '';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.slider.showSlide(function(){
					eventBus.$emit('init-config','add', vm.paramModelClickData.param_model,'');
				});
			},
			//下载模板
			exportTemplate(){
				var url = "${ctx}/dataModel/enb/standardPrivateRela/downloadImportDataModelTemplate.action";
				exportByForm(url, {});
			},
			
			//import param_model 
			importParamModelBtn(){
				var vm = this;
				vm.importParamShow = true;
				vm.addProTypeShow = false;
				vm.addProductModelShow = false;
				vm.importProductTypeErrorMessage = '';
				vm.importProductTypeMessage = '';
				vm.curSelectProductModelData = [];
				vm.$refs.productTypeTable.clearSelection();
			},
			//import param_model: add product type manually
			addProductTypeBtn(){
				var vm = this, productModelText = vm.importParamForm.productModelStr, str = '';
				if(productModelText == ''){
					vm.importProductTypeErrorMessage = '<%=rb.getString("QingShuRuChanPinLeiXing")%>';
				}else{
					if(productModelText){
						str = productModelText;
						if(vm.importParamForm.productTypeList.indexOf(str) == -1){
							vm.importParamForm.productTypeList.push(str);
							vm.importParamForm.productModelStr = '';
							vm.importProductTypeErrorMessage = '';
							vm.importProductTypeMessage = '';
							vm.$refs.importParamForm.validateField('itemTest');
						}else{
							vm.importProductTypeErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.importProductTypeErrorMessage = '<%=rb.getString("QingShuRuChanPinLeiXing")%>';
					}
				}
			},
			//import param_model: manually delect product type 
			removeProductType(item){
				var vm = this;
				var index = vm.importParamForm.productTypeList.indexOf(item);
				if(index !== -1){
					vm.importParamForm.productTypeList.splice(index,1)
				}
				vm.importProductTypeErrorMessage = '';
			},
			//import param_model: product type list selection
			productModelBatchSelect(selection){
				var vm = this; 
				if(selection.length > 0){
					vm.curSelectProductModelData = selection;
					vm.importProductTypeMessage = '';
				}else{
					vm.curSelectProductModelData = [];
				}
			},
			//发送请求，校验device文件内容 
			checkFile(res,file){    
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.importParamShow = false;
					vm.$refs.paramModelTable.refresh();
					vm.closeFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.upload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			fileChange(file,fileList){ 
				var vm = this;
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
			},
			// 选择文件
			fileSelect(){  
				var vm =this;
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			// 移除导入文件
			closeFileSelect(){
				var vm = this;
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this,
				fileName = file.name,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
				fd.append('uploadFile',file); 
				fd.append('paramModel',vm.importParamForm.paramModel);
				fd.append('productTypes',vm.mergeProductTypelList.join(','));
				axios.post("${ctx}/dataModel/enb/standardPrivateRela/uploadInfoForParamModel.action",fd,config).then(function(res){
					if(res.data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.paramModelTable.refresh();
						vm.$refs.paramList.refresh();
						vm.importParamShow = false;
						vm.paramModel = '';
						vm.fileList = [];
						vm.fileName = '';
						vm.$refs.importParamForm.resetFields();						
					}else{
						vm.$message.error(res.data["message"])
					}
				})
				
				return false;
			},
			//这里才是导入真正的提交，需要走接口验证 rtd是否存在
			confirmImportParamModelExists(){
				var vm = this;
				vm.$refs.upload.submit();
			},
			//关闭 param_model 已存在弹窗
			cancelImportParamModelExists(){
				var vm = this;
				vm.importParamModelExistsDialog = false;
				vm.importParamCancel();
			},
			
			//import param_model: submit
			importParamSubmit(){
				var vm = this, addflag = false,
					manualProductTypeList = vm.importParamForm.productTypeList,
					curModelList = vm.curSelectProductModelData.map(function(item){return item.productType });  
				
    			if(manualProductTypeList.length == 0 && vm.curSelectProductModelData.length == 0){
        			vm.importProductTypeMessage = '<%=rb.getString("ChanPinLeiXingZhiShaoTianJiaYiGe")%>';
        			addflag = true;
        		}else{        		
        			vm.importProductTypeMessage = '';
        			addflag = false;
        			//数组合并 去重
            		var mergelist = manualProductTypeList.concat(curModelList);
            		for(var i =0, len = mergelist.length; i<len; i++){
            			if(vm.mergeProductTypelList.indexOf(mergelist[i]) === -1){
            				vm.mergeProductTypelList.push(mergelist[i])
            			}
            		}
        		}
				
				vm.$refs.importParamForm.validate((valid) => {
                    if (valid && addflag == false) {
                    	//校验 param_model 是否已存在
                    	axios.get("${ctx}/dataModel/enb/standardPrivateRela/verifyParamModelExist.action",stringify({paramModel: vm.importParamForm.paramModel})).then(function(response){
	    					//var data = true;
	    					var data = response.data;
	    					if(data){
	    						//已存在
	    						vm.existsParamModel = vm.importParamForm.paramModel;
	    						vm.importParamModelExistsDialog = true;
	    					}else{
	    						//不存在
	    						vm.$refs.upload.submit();
	    					}
    					})                    	
                    }
                }) 
			},
			//import param_model: cancel 
			importParamCancel(){
				var vm = this;
				vm.importParamShow = false;
				vm.fileList = [];
				vm.fileName = '';
				vm.importProductTypeErrorMessage = '';
				vm.importProductTypeMessage = '';
				vm.curSelectProductModelData = [];
				vm.$refs.productTypeTable.clearSelection();
				vm.$refs.importParamForm.resetFields();
			},
			//确定删除
			delSubmit(){
				var vm = this;
				
				if(vm.curDelTtype == 'modelNameDel'){
					var params = {
						id: vm.rowData.id,
						productClass: vm.rowData.productClass //新增参数
					};
					axios.post("${ctx}/dataModel/enb/productModelProductNameRela/deleteInfo.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.modelNameTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
						vm.delDialog = false;
					})
				}else if(vm.curDelTtype == 'paramModelDel'){
					var params = {
						paramModel: vm.paramModelRowData.param_model
					};
					axios.post("${ctx}/dataModel/enb/standardPrivateRela/deleteInfoForParamModel.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.paramModelTable.refresh();
                            vm.$refs.paramList.refresh();
						}else{
							vm.$message.error(data["message"])
						}
						vm.delDialog = false;
					})
				}else if(vm.curDelTtype == 'paramListDel'){
					var params = {
							id: vm.paramListClickData.id
						};
					axios.post("${ctx}/dataModel/enb/standardPrivateRela/deleteInfoForId.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.paramModelTable.refresh();
                            vm.$refs.paramList.refresh();
						}else{
							vm.$message.error(data["message"])
						}
						vm.delDialog = false;
					})
				}
			},
			///----------------------------------param model
			//param_model 表格加载成功
			tableLoadSuccess(req){
				var vm = this, rows = req.rows, row = '';
				if(rows && rows.length) {
					row = rows[0];
					vm.paramModelClickData = row;
					vm.parmParams.paramModel = row.param_model;//默认选中第一条数据
					vm.$refs.paramModelTable.setCurrentRow(row);
				}
			},
			//param_model 表格操作
			groupOpClick(row,ev){
				var vm = this, paramModelDelFlag = false;
				
				vm.paramModelRowData = row;
				//is_build_in  1- 不可修改, 删除； 0 -可修改，删除
    			if(row.is_build_in == '1'){
    				paramModelDelFlag = true;
    			}else{
    				paramModelDelFlag = false;
    			}
				
				vm.menusGroup= [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info', row: row},
					{label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download",code:'download', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del', row: row, disable: paramModelDelFlag}
				];
				vm.$nextTick(function(){
		    		document.body.click();
		    		vm.$refs.menuGroup.show(ev);
		    	});
			},
			clickMenu(ev){ 
				var vm = this,
				
				codes = {
					info: vm.infoParam,
					download: vm.downloadParam,
					del: vm.deleteParam,
				};
				if(codes[ev.code]){
					codes[ev.code](vm.paramModelRowData)
				}
			},
			//param_model 详情
			infoParam(row){
				var vm = this;
				if(row){
					vm.paramModelInfo = row.param_model;
					if(row.product_type === '' || row.product_type === null || row.product_type === undefined){
						vm.productTypeInfo = [];
					}else{
						vm.productTypeInfo = row.product_type.split(',');
					}
				}
				
				vm.paramModelInfoDialog = true;
			},
			//param_model 详情关闭
			cancelInfo(){
				var vm = this;
				
				vm.paramModelInfoDialog = false;
			},
			//param_model 下载
			downloadParam(row){
				var vm = this, 
				params = {
					paramModel: row.param_model	
				}
				var url = "${ctx}/dataModel/enb/standardPrivateRela/downloadDataModel.action";
				exportByForm(url, params);
			},
			//param_model 删除
			deleteParam(row){
				var vm = this;
				vm.delDialog = true;
				vm.importParamShow = false;
				vm.curDelTtype = 'paramModelDel';
			},
			//右侧选中事件
			paramModelClick(currentRow,oldCurrentRow){
				var vm = this;
				if(currentRow !== undefined){
					vm.paramModelClickData = currentRow;
					//默认选中第一条数据
					vm.parmParams.paramModel = currentRow.param_model;
				}else{
					vm.paramModelClickData = []
				}
			},
			// param list 表格操作
			paramListClick(row,ev){
				var vm = this, paramDelFlag = false;
			
				vm.paramListClickData = row;
				//is_build_in  1- 不可修改, 删除； 0 -可修改，删除
    			if(row.is_build_in == '1'){
    				paramDelFlag = true;
    			}else{
    				paramDelFlag = false;
    			}
				
				vm.menusParamList= [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info', row: row},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit",code:'modify', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del', row: row, disable: paramDelFlag}
				];
				vm.$nextTick(function(){
		    		document.body.click();
		    		vm.$refs.menuParamList.show(ev);
		    	});
			},
			//param list 表格操作项
			clickParamListMenu(ev){
				var vm = this,
				
				codes = {
					info: vm.infoParamList,
					modify: vm.modifyParamList,
					del: vm.deleteParamList
				};
				if(codes[ev.code]){
					codes[ev.code](vm.paramListClickData)
				}
			},
			//param info
			infoParamList(row){
				var vm = this,
				url = '${ctx}/dataModel/enb/standardPrivateRela/goAddDataModelJSPPage.action';
				vm.slideTitle = '';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.slider.showSlide(function(){
					eventBus.$emit('init-config','readonly', vm.paramModelClickData.param_model, row);
				});
			},
			//param modify
			modifyParamList(row){
				var vm = this,
				url = '${ctx}/dataModel/enb/standardPrivateRela/goAddDataModelJSPPage.action';
				vm.slideTitle = '';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.slider.showSlide(function(){
					eventBus.$emit('init-config','modify', vm.paramModelClickData.param_model, row);
				});
			},
			//param delete
			deleteParamList(row){
				var vm = this;	
				vm.delDialog = true;
				vm.curDelTtype = 'paramListDel';
			},
			//删除 弹窗-取消操作
			cancelDel(){
				var vm = this;
				vm.delDialog = false;
			},
			// table menu hide
			modelHanderClose() {
				var vm = this;
				vm.$refs.productModelMenu.hide();
				vm.$refs.menuGroup.hide();
				vm.$refs.menuParamList.hide();
			},				
			addConfig() {
				eventBus.$emit('save-config');
			},		
			closeConfig() {
				var vm = this;				
				eventBus.$emit('hander-cancel')
			},
			//refresh  
			reloadConfig() {
				var vm = this;
				vm.$refs.paramModelTable.refresh();
				vm.$refs.paramList.refresh();
			},
			//设备类型：model name list search
			queryModelName(text) {
				this.modelNameParam.search_text = text;
			},
			//数据模型：param list search
			queryParamPath(text){
				this.parmParams.search_text = text;
			},
		    // slide close
		    cancelSlide(){
		    	var vm = this;
				vm.$refs.slider.hide();
			},
		},
		mounted() {
			eventBus.$off('close-config').$on('close-config', this.closeConfig);
			eventBus.$off('reload-config-list').$on('reload-config-list', this.reloadConfig);
			eventBus.$off('cancel-slide').$on('cancel-slide',this.cancelSlide);
		}
	});

</script>